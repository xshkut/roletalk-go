package roletalk

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	errors "github.com/pkg/errors"

	"github.com/gorilla/websocket"
)

//Peer is the local node of your peer-to-peer architecture
type Peer struct {
	Name            string
	id              string
	Friendly        bool
	units           map[string]*Unit
	roles           map[string]*Role
	destinations    map[string]*Destination
	addrUnits       *addressScheme
	servers         []*http.Server
	alive           sync.WaitGroup
	presharedKeys   []PresharedKey
	incMsgChan      chan *MessageContext
	startTime       time.Time
	roleRWMutex     sync.RWMutex
	destRWMutex     sync.RWMutex
	unitRWMutex     sync.RWMutex
	closeHandlers   []func()
	unitHandlers    []unitHandler
	roleHandlers    []roleHandler
	lastRolesChange int
}

//NewPeer creates Peer, e.g. local node in your peer-to-peer architecture
func NewPeer(opts PeerOptions) *Peer {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	name := opts.Name
	friendly := opts.Friendly

	peer := &Peer{id: uuid, Name: name, Friendly: friendly, startTime: time.Now()}
	peer.incMsgChan = make(chan *MessageContext)
	peer.destinations = make(map[string]*Destination)
	peer.roles = make(map[string]*Role)
	peer.units = make(map[string]*Unit)
	peer.addrUnits = newAddressScheme()

	for i := 0; i < runtime.NumCPU(); i++ {
		go peer.consumeIncomingMessages()
	}

	return peer
}

//Connect establishes connection to remote peer and creates Unit
func (peer *Peer) Connect(urlStr string, opts ...ConnectOptions) (unit *Unit, err error) {
	var conn *connLocker

	options := ConnectOptions{}
	if len(opts) > 0 {
		options = opts[0]
	}

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: options.InsecureTLS}

	if options.DoNotReconnect == false {
		peer.addrUnits.store(urlStr, nil, nil)
	}

	c, _, err := dialer.Dial(urlStr, http.Header{})
	if err != nil {
		err = errors.Wrap(err, "Cannot dial to remote peer")
		goto errCase
	}

	conn = createConnLocker(c)
	unit, err = peer.addConn(conn)
	if err != nil {
		err = errors.Wrap(err, "Cannot use connection to communicate")
		goto errCase
	}

	if options.DoNotReconnect == false {
		peer.addrUnits.store(urlStr, conn, unit)
	}
	if options.DoNotAcquaint == false {
		go peer.introduceUnitToOthers(unit, urlStr)
		go unit.acquaintWithOthers()
	}

	return

errCase:
	if options.DoNotReconnect == false {
		go peer.startReconnCycle(urlStr, true)
	}
	return nil, err
}

//ListRoles returns all registered roles as list []string
func (peer *Peer) ListRoles() []string {
	peer.roleRWMutex.RLock()
	sl := make([]string, len(peer.roles))
	i := 0
	for key, role := range peer.roles {
		if role.Active() == false {
			continue
		}
		sl[i] = key
		i++
	}
	peer.roleRWMutex.RUnlock()
	return sl[0:i]
}

//ListDestinations returns all registered destinations as list []string
func (peer *Peer) ListDestinations() []string {
	peer.destRWMutex.RLock()
	sl := make([]string, len(peer.destinations))
	i := 0
	for key := range peer.destinations {
		sl[i] = key
		i++
	}
	peer.destRWMutex.RUnlock()
	return sl
}

//Role creates or retrieves Role with provided name
func (peer *Peer) Role(name string) *Role {
	var role *Role
	var ok bool
	if role, ok = peer.getRole(name); ok == false {
		role = peer.createRole(name)
		peer.roleRWMutex.Lock()
		peer.roles[name] = role
		peer.roleRWMutex.Unlock()
		go peer.runOnRole(role)
		go peer.broadcastRoles()
	}
	return role
}

//Destination returns Destination with provided name or first creates it if such does not exist
//Destination prepresents corresponding remote peers' roles
func (peer *Peer) Destination(name string) *Destination {
	peer.destRWMutex.Lock()
	defer peer.destRWMutex.Unlock()

	if _, ok := peer.destinations[name]; ok == false {
		dest := peer.createDestination(name)
		peer.destinations[name] = dest

		peer.unitRWMutex.RLock()
		for _, unit := range peer.units {
			unit.rolesMx.RLock()
			if _, ok := unit.roles[name]; ok == true {
				dest.addUnit(unit)
			}
			unit.rolesMx.RUnlock()
		}
		peer.unitRWMutex.RUnlock()

	}
	return peer.destinations[name]
}

//Units returns map for all connected units of the Peer
func (peer *Peer) Units() []*Unit {
	peer.unitRWMutex.RLock()
	defer peer.unitRWMutex.RUnlock()
	units := make([]*Unit, len(peer.units))
	i := 0
	for _, val := range peer.units {
		units[i] = val
		i++
	}
	return units
}

//Unit returns referrence to Unit with provided id or nil if such does not exist or is not connected
func (peer *Peer) Unit(id string) *Unit {
	peer.unitRWMutex.RLock()
	unit := peer.units[id]
	peer.unitRWMutex.RUnlock()
	return unit
}

func (peer *Peer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{} // use default options
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade: ", err)
		return
	}
	peer.InvolveConn(c)
}

//Listen to incoming connections
func (peer *Peer) Listen(address string) (net.Addr, error) {
	server := &http.Server{Handler: peer, Addr: address}
	peer.servers = append(peer.servers, server)
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return nil, err
	}
	peer.alive.Add(1)
	go func() {
		defer peer.alive.Done()
		if err := server.Serve(listener); err != http.ErrServerClosed {
			panic(err)
		}
	}()
	return listener.Addr(), nil
}

//Close all listeners and connections
func (peer *Peer) Close() {
	for _, server := range peer.servers {
		server.Close()
	}
}

//WaitForClose waits until all units (their underlying connections) and listeners will be closed. Could be used to prevent Main() from returning
func (peer *Peer) WaitForClose() {
	peer.alive.Wait()
}

//OnUnit adds unit handler function f which executes synchronosly with other unit handlers of the Peer in FIFO order when Peer gets new Unit
func (peer *Peer) OnUnit(f func(unit *Unit)) {
	peer.unitRWMutex.Lock()
	peer.unitHandlers = append(peer.unitHandlers, f)
	peer.unitRWMutex.Unlock()
}

//OnRole adds role handler function f which executes synchronosly with other role handlers of the Peer in FIFO order when Peer gets new Role
func (peer *Peer) OnRole(f func(role *Role)) {
	peer.roleRWMutex.Lock()
	peer.roleHandlers = append(peer.roleHandlers, f)
	peer.roleRWMutex.Unlock()
}

//ID returns peer's unique identificator. It is created when peer is constructed.
func (peer *Peer) ID() string {
	return peer.id
}

//ConnectOptions specifies options for outgoing connection
type ConnectOptions struct {
	DoNotReconnect bool //set true if connection is not supposed to reconnect after abort
	DoNotAcquaint  bool //set true if connection is not supposed to be introduced to remote peers nor to be acquainted with remote peers' by their addresses
	InsecureTLS    bool //set true if TLS errors are supposed to be ignored for outgoing connection
}
