package roletalk

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type unitsMap map[*Unit]interface{}

func (peer *Peer) createDestination(name string) *Destination {
	return &Destination{
		name:       name,
		peer:       peer,
		ready:      false,
		units:      make(map[*Unit]interface{}),
		stateMutex: sync.RWMutex{},
	}
}

func (dest *Destination) addUnit(unit *Unit) {
	if dest.HasUnit(unit) {
		return
	}
	dest.stateMutex.Lock()
	dest.units[unit] = struct{}{}
	if len(dest.units) == 1 {
		dest.ready = true
		go dest.runOnUnit(unit)
	}
	dest.stateMutex.Unlock()
}

func (dest *Destination) deleteUnit(unit *Unit) {
	if dest.HasUnit(unit) {
		dest.stateMutex.Lock()
		delete(dest.units, unit)
		if len(dest.units) < 1 {
			dest.ready = false
			go dest.runOnClose()
		}
		dest.stateMutex.Unlock()
	}
}

func (dest *Destination) runOnClose() {
	dest.stateMutex.RLock()
	handlers := dest.closeHandlers
	dest.stateMutex.RUnlock()
	for _, handler := range handlers {
		handler()
	}
}

func (dest *Destination) runOnUnit(unit *Unit) {
	dest.stateMutex.RLock()
	handlers := dest.unitHandlers
	dest.stateMutex.RUnlock()
	for _, handler := range handlers {
		handler(unit)
	}
}

func (dest *Destination) nextUnit() (*Unit, error) {
	i := atomic.AddUint32(&dest.unitIndex, 1)
	units := dest.Units()
	l := len(units)
	if l < 1 {
		return nil, fmt.Errorf("No units connected to serve role %v", dest.name)
	}
	return units[i%uint32(l)], nil
}
