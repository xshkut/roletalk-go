package roletalk

import (
	"time"

	"github.com/gorilla/websocket"
)

func (peer *Peer) getRole(name string) (*Role, bool) {
	peer.roleRWMutex.RLock()
	role, ok := peer.roles[name]
	peer.roleRWMutex.RUnlock()
	return role, ok
}

func (peer *Peer) broadcastRoles() {
	roles := peer.ListRoles()
	units := peer.Units()
	// fmt.Printf("Sending roles from %v, %v\n", peer.ID(), roles)
	for _, unit := range units {
		err := unit.sendRoles(roles)
		_ = err
	}
}

func (peer *Peer) createUnit(res peerData) *Unit {
	unit := Unit{id: res.ID, friendly: res.Friendly, meta: res.Meta, name: res.Name, peer: peer}
	unit.roles = make(map[string]interface{})
	for _, role := range res.Roles {
		unit.roles[role] = struct{}{}
	}
	unit.callbackCtr = createCallbackController()
	unit.streamCtr = *createStreamController()
	return &unit
}

func (peer *Peer) addConn(conn *connLocker) (unit *Unit, err error) {
	res, err := peer.authenticateWS(conn)
	if err != nil {
		closeConnWithCode(conn, errAuthRejected, "auth rejected")
		return
	}
	if err := checkProtocolCompatibility(protocolVersion, res.Meta.Protocol); err != nil {
		closeConnWithCode(conn, errIncompatibleProtocolVersion, err.Error())
		return nil, err
	}
	_, unitExists := peer.getUnit(res.ID)
	if unitExists == false {
		peer.addUnit(peer.createUnit(res))
	}
	unit, _ = peer.getUnit(res.ID)
	if unitExists == false {
		peer.runOnUnit(unit)
	}
	peer.onNewUnitRoles(unit)
	unit.bindConn(conn)
	return
}

func (peer *Peer) onNewUnitRoles(unit *Unit) {
	var inDest bool
	// fmt.Printf("receiving roles from %v, %v\n", unit.ID(), peer.ID())
	for name, dest := range peer.destinations {
		inDest = false
		unit.rolesMx.RLock()
		roles := unit.roles
		unit.rolesMx.RUnlock()
		for role := range roles {
			if name == role {
				dest.addUnit(unit)
				inDest = true
				break
			}
		}
		if inDest == false {
			dest.deleteUnit(unit)
		}
	}
}

func (peer *Peer) deleteUnit(u *Unit, err error) {
	peer.unitRWMutex.Lock()
	delete(peer.units, u.id)
	peer.destRWMutex.Lock()
	for r := range u.roles {
		dest, ok := peer.destinations[r]
		if ok == false {
			break
		}
		dest.deleteUnit(u)
	}
	peer.destRWMutex.Unlock()
	peer.unitRWMutex.Unlock()
	go u.runOnClose(err)
}

func (peer *Peer) addUnit(u *Unit) {
	peer.unitRWMutex.Lock()
	peer.units[u.id] = u
	peer.unitRWMutex.Unlock()
}

func (peer *Peer) getUnit(id string) (*Unit, bool) {
	peer.unitRWMutex.RLock()
	u, ok := peer.units[id]
	peer.unitRWMutex.RUnlock()
	return u, ok
}

func (peer *Peer) startReconnCycle(addr string, waitFirst bool) {
	if waitFirst {
		time.Sleep(reconnInterval)
	}

	for {
		uc, ok := peer.addrUnits.loadByAddress(addr)
		if uc.conn != nil || ok == false {
			return
		}

		_, err := peer.Connect(addr, ConnectOptions{DoNotReconnect: true})
		if err == nil {
			return
		}

		time.Sleep(reconnInterval)
	}
}

func (peer *Peer) runOnUnit(unit *Unit) {
	peer.unitRWMutex.RLock()
	handlers := peer.unitHandlers
	peer.unitRWMutex.RUnlock()
	for _, handler := range handlers {
		handler(unit)
	}
}

func (peer *Peer) runOnRole(role *Role) {
	peer.roleRWMutex.RLock()
	handlers := peer.roleHandlers
	peer.roleRWMutex.RUnlock()
	for _, handler := range handlers {
		handler(role)
	}
}

//InvolveConn accepts conn for authentication and communication
func (peer *Peer) InvolveConn(c *websocket.Conn) (*Unit, error) {
	conn := createConnLocker(c)
	return peer.addConn(conn)
}
