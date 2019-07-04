package roletalk

import (
	"sync"
	"time"
)

//Destination represents a role (service name) of remote peers (units).
//Destination is used as gateway for outgoing communication. It implements round-robin client-site load balancing. To communicate with specific remote peer (unit) use EmitOptions
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

//Name returns name of destination. It represents the name of corresponding role of remote peers (units)
func (dest *Destination) Name() string {
	dest.stateMutex.RLock()
	name := dest.name
	dest.stateMutex.RUnlock()
	return name
}

//HasUnit returns true if provided unit serves a role with a name corresponding to the destination
func (dest *Destination) HasUnit(unit *Unit) bool {
	dest.stateMutex.RLock()
	_, ok := dest.units[unit]
	dest.stateMutex.RUnlock()
	return ok
}

//Units returns slice of all units which serve the role
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

//Ready indicates whether destination has connected units or not
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

//Request sends request message to remote peer (Unit). Returns error if remote peer rejected the request or request timed out
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

//NewReader sends request to remote peer (Unit) to establish stream session. This end of the stream is readable.
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

//NewWriter sends request to remote peer (Unit) to establish stream session. This end of the stream is readable.
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

//OnClose adds handler function f which runs synchronosly with other close handlers of the destination in FIFO order when Destination losts last unit
func (dest *Destination) OnClose(f func()) {
	dest.stateMutex.Lock()
	dest.closeHandlers = append(dest.closeHandlers, f)
	dest.stateMutex.Unlock()
}

//OnUnit adds handler function f which executes synchronosly with other unit handlers of the destination in FIFO order when Destination gets new unit
func (dest *Destination) OnUnit(f func(unit *Unit)) {
	dest.stateMutex.Lock()
	dest.unitHandlers = append(dest.unitHandlers, f)
	dest.stateMutex.Unlock()
}

//EmitOptions determines Data to send and options for transferring it. All fields are optional
//specify Unit to send data to; Timeout for callback (ignored for Send and Broadcast);
type EmitOptions struct {
	Data            interface{}
	Unit            *Unit
	Timeout         time.Duration
	IgnoreUnitClose bool
}
