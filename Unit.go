package roletalk

import (
	"sync"
)

//Unit represents remote peer
type Unit struct {
	peer          *Peer
	id            string
	name          string
	friendly      bool
	meta          MetaInfo
	roles         map[string]interface{}
	rolesMx       sync.RWMutex
	connections   sync.Map
	connsMx       sync.RWMutex
	streamCtr     streamController
	callbackCtr   reqCallbackController
	closeHandlers []func(err error)
}

//Close all underlying connections
func (unit *Unit) Close() {
	unit.peer.addrUnits.deleteUnit(unit)
	unit.callbackCtr.onClose()
	unit.closeWithCode(errManualClose, "method .Close() called")
}

//GetRoles returns list of roles the unit serves
func (unit *Unit) GetRoles() []string {
	unit.rolesMx.RLock()
	sl := make([]string, len(unit.roles))
	i := 0
	for role := range unit.roles {
		sl[i] = role
		i++
	}
	unit.rolesMx.RUnlock()
	return sl
}

//HasRole returns true if unit serves the role with provided name
func (unit *Unit) HasRole(name string) bool {
	unit.rolesMx.RLock()
	_, has := unit.roles[name]
	unit.rolesMx.RUnlock()
	return has
}

//Connected returns true if unit's underlying conn's were not been closed.
//If all conns's are closed or there was at least a moment when all conn's were closed, Connected returns false
func (unit *Unit) Connected() bool {
	u := unit.peer.Unit(unit.id)
	return u == unit
}

//Name returns remote peer's name
func (unit *Unit) Name() string {
	return unit.name
}

//ID returns remote peer's id
func (unit *Unit) ID() string {
	return unit.id
}

//Friendly indicates whether remote peer handles acquaint messages
func (unit *Unit) Friendly() bool {
	return unit.friendly
}

//Meta returns unit's meta data
func (unit *Unit) Meta() MetaInfo {
	return unit.meta
}

//OnClose adds handler function f which runs synchronosly with other close handlers of the destination in FIFO order when Destination losts last unit
func (unit *Unit) OnClose(f func(err error)) {
	unit.connsMx.Lock()
	unit.closeHandlers = append(unit.closeHandlers, f)
	unit.connsMx.Unlock()
}
