package roletalk

import "sync"

type addressScheme struct {
	mx        sync.RWMutex
	addresses map[string]*unitConn
	conns     map[*connLocker]string
}

type unitConn struct {
	unit *Unit
	conn *connLocker
}

func newAddressScheme() *addressScheme {
	ca := addressScheme{}
	ca.addresses = make(map[string]*unitConn)
	ca.conns = make(map[*connLocker]string)
	return &ca
}

func (ca *addressScheme) loadByAddress(address string) (*unitConn, bool) {
	ca.mx.RLock()
	u, ok := ca.addresses[address]
	ca.mx.RUnlock()
	return u, ok
}

func (ca *addressScheme) store(address string, conn *connLocker, u *Unit) {
	ca.mx.Lock()
	ca.addresses[address] = &unitConn{u, conn}
	ca.conns[conn] = address
	ca.mx.Unlock()
}

func (ca *addressScheme) loadByConn(c *connLocker) (str string, ok bool) {
	ca.mx.RLock()
	str, ok = ca.conns[c]
	ca.mx.RUnlock()
	return
}

func (ca *addressScheme) unbindAddr(address string) {
	ca.mx.Lock()
	uc, ok := ca.addresses[address]
	if ok == true {
		ca.addresses[address] = nil
		delete(ca.conns, uc.conn)
	}
	ca.mx.Unlock()
}

func (ca *addressScheme) deleteAddr(address string) {
	ca.mx.Lock()
	if uc, ok := ca.addresses[address]; ok == true {
		delete(ca.conns, uc.conn)
	}
	delete(ca.addresses, address)
	ca.mx.Unlock()
}

func (ca *addressScheme) deleteUnit(u *Unit) {
	ca.mx.Lock()
	for addr, uc := range ca.addresses {
		if uc.unit == u {
			delete(ca.addresses, addr)
			delete(ca.conns, uc.conn)
		}
	}
	ca.mx.Unlock()
}

func (ca *addressScheme) getUnitMap() map[string]*Unit {
	ca.mx.RLock()
	m := make(map[string]*Unit)
	for addr, uc := range ca.addresses {
		if uc == nil {
			continue
		}
		m[addr] = uc.unit
	}
	ca.mx.RUnlock()
	return m
}
