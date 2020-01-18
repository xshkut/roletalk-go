package roletalk

import (
	"sync"
	"time"
)

//Destination represents a role (a service name) of remote peers (units).
//Destination is used as a gateway for outgoing communication. It implements round-robin load balancing between units. To communicate with specific remote peer (unit) use EmitOptions
type Destination struct {
	name          string
	peer          *Peer
	ready         bool
	units         unitsMap
	stateMutex    sync.RWMutex
	unitIndex     uint32
	closeHandlers []func()
	unitHandlers  []unitHandler
}

//Name returns Destination's name
func (dest *Destination) Name() string {
	dest.stateMutex.RLock()
	name := dest.name
	dest.stateMutex.RUnlock()
	return name
}

//HasUnit returns true if provided unit serves a role with Destination's name
func (dest *Destination) HasUnit(unit *Unit) bool {
	dest.stateMutex.RLock()
	_, ok := dest.units[unit]
	dest.stateMutex.RUnlock()
	return ok
}

//Units returns slice of all connected units serving corresponding role
func (dest *Destination) Units() []*Unit {
	dest.stateMutex.RLock()
	un := make([]*Unit, len(dest.units))
	i := 0
	for unit := range dest.units {
		un[i] = unit
		i++
	}
	dest.stateMutex.RUnlock()
	return un
}

//Ready indicates whether Destination has connected units
func (dest *Destination) Ready() bool {
	dest.stateMutex.RLock()
	ready := dest.ready
	dest.stateMutex.RUnlock()
	return ready
}

//Send sends one-way message to remote peer (Unit). Returns error if message has not been written to underlying connection
func (dest *Destination) Send(event string, opts EmitOptions) error {
	var unit *Unit
	var err error
	if opts.Unit != nil {
		unit = opts.Unit
	} else {
		if unit, err = dest.nextUnit(); err != nil {
			return err
		}
	}
	return unit.send(emitStruct{event: event, role: dest.name, timeout: opts.Timeout, data: opts.Data})
}

//Request emits request message to remote peer (Unit). Returns error if remote peer rejected the request or request timed out, otherwise returns response context
func (dest *Destination) Request(event string, opts EmitOptions) (res *MessageContext, err error) {
	var unit *Unit
	if opts.Unit != nil {
		unit = opts.Unit
	} else {
		if unit, err = dest.nextUnit(); err != nil {
			return
		}
	}
	return unit.request(emitStruct{event: event, role: dest.name, timeout: opts.Timeout, data: opts.Data, ignoreUnitClose: opts.IgnoreUnitClose})
}

//NewReader requests for creating binary stream session and returns its readable end.
//Returns error if remote peer rejected the request or request timed out
func (dest *Destination) NewReader(event string, opts EmitOptions) (res *MessageContext, r *Readable, err error) {
	var unit *Unit
	if opts.Unit != nil {
		unit = opts.Unit
	} else {
		if unit, err = dest.nextUnit(); err != nil {
			return
		}
	}
	return unit.newReader(emitStruct{event: event, role: dest.name, timeout: opts.Timeout, data: opts.Data, ignoreUnitClose: opts.IgnoreUnitClose})
}

//NewWriter requests for creating binary stream session and returns its writable end.
//Returns error if remote peer rejected the request or request timed out
func (dest *Destination) NewWriter(event string, opts EmitOptions) (res *MessageContext, writable *Writable, err error) {
	var unit *Unit
	if opts.Unit != nil {
		unit = opts.Unit
	} else {
		if unit, err = dest.nextUnit(); err != nil {
			return
		}
	}
	return unit.newWriter(emitStruct{event: event, role: dest.name, timeout: opts.Timeout, data: opts.Data, ignoreUnitClose: opts.IgnoreUnitClose})
}

//OnClose adds handler function f which runs synchronosly with other close handlers in FIFO order when last Unit gets disconnected
func (dest *Destination) OnClose(f func()) {
	dest.stateMutex.Lock()
	dest.closeHandlers = append(dest.closeHandlers, f)
	dest.stateMutex.Unlock()
}

//OnUnit adds handler function f which executes synchronosly with other unit handlers in FIFO order when it gets new unit
func (dest *Destination) OnUnit(f func(unit *Unit)) {
	dest.stateMutex.Lock()
	dest.unitHandlers = append(dest.unitHandlers, f)
	dest.stateMutex.Unlock()
}

//EmitOptions determines Data to send and additional transfer options. All fields are optional.
//Specify Unit to send data to; Timeout for callback (Timeout option is ignored for Send and Broadcast methods);
//If IgnoreUnitClose is true, request will not be rejected internally when communicated unit disconnects, but timeout will still has its place.
type EmitOptions struct {
	Data            interface{}
	Unit            *Unit
	Timeout         time.Duration
	IgnoreUnitClose bool
}
