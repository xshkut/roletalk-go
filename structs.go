package roletalk

import (
	"sync"
)

//PresharedKey is used for authentication. id is used to identify the key
type PresharedKey struct {
	id  string
	key string
}

//PeerOptions provide options to create Peer
type PeerOptions struct {
	Name     string
	Friendly bool
}

//ConnectOptions specifies options for outgoing connection
type ConnectOptions struct {
	Weak bool //set true if connection is not supposed to reconnect after abort. Weak connection is not introduced to other friendly units
}

type middlewareMessageMap struct {
	mx sync.RWMutex
	m  map[string][]MessageHandler
}

type unitHandler func(u *Unit)
type roleHandler func(r *Role)

func createMiddlewareMessageMap() *middlewareMessageMap {
	return &middlewareMessageMap{m: make(map[string][]MessageHandler)}
}

func (roleMW *middlewareMessageMap) get(event string) []MessageHandler {
	roleMW.mx.RLock()
	handlers, ok := roleMW.m[event]
	if ok == false {
		handlers = []MessageHandler{}
	}
	roleMW.mx.RUnlock()
	return handlers
}

func (roleMW *middlewareMessageMap) set(event string, handler MessageHandler) {
	roleMW.mx.Lock()
	if _, ok := roleMW.m[event]; ok == false {
		roleMW.m[event] = []MessageHandler{}
	}
	roleMW.m[event] = append(roleMW.m[event], handler)
	roleMW.mx.Unlock()
}

type middlewareRequestMap struct {
	mx sync.RWMutex
	m  map[string][]RequestHandler
}

type middlewareReaderRequestMap struct {
	mx sync.RWMutex
	m  map[string][]ReadableRequestHandler
}
type middlewareWriterRequestMap struct {
	mx sync.RWMutex
	m  map[string][]WritableRequestHandler
}

func createMiddlewareRequestMap() *middlewareRequestMap {
	return &middlewareRequestMap{m: make(map[string][]RequestHandler)}
}

func createMiddlewareReaderMap() *middlewareReaderRequestMap {
	return &middlewareReaderRequestMap{m: make(map[string][]ReadableRequestHandler)}
}

func createMiddlewareWriterMap() *middlewareWriterRequestMap {
	return &middlewareWriterRequestMap{m: make(map[string][]WritableRequestHandler)}
}

func (roleMW *middlewareRequestMap) get(event string) []RequestHandler {
	roleMW.mx.RLock()
	handlers, ok := roleMW.m[event]
	if ok == false {
		handlers = []RequestHandler{}
	}
	roleMW.mx.RUnlock()
	return handlers
}

func (roleMW *middlewareRequestMap) set(event string, handler RequestHandler) {
	roleMW.mx.Lock()
	if _, ok := roleMW.m[event]; ok == false {
		roleMW.m[event] = []RequestHandler{}
	}
	roleMW.m[event] = append(roleMW.m[event], handler)
	roleMW.mx.Unlock()
}

func (roleMW *middlewareWriterRequestMap) get(event string) []WritableRequestHandler {
	roleMW.mx.RLock()
	handlers, ok := roleMW.m[event]
	if ok == false {
		handlers = []WritableRequestHandler{}
	}
	roleMW.mx.RUnlock()
	return handlers
}

func (roleMW *middlewareWriterRequestMap) set(event string, handler WritableRequestHandler) {
	roleMW.mx.Lock()
	if _, ok := roleMW.m[event]; ok == false {
		roleMW.m[event] = []WritableRequestHandler{}
	}
	roleMW.m[event] = append(roleMW.m[event], handler)
	roleMW.mx.Unlock()
}

func (roleMW *middlewareReaderRequestMap) get(event string) []ReadableRequestHandler {
	roleMW.mx.RLock()
	handlers, ok := roleMW.m[event]
	if ok == false {
		handlers = []ReadableRequestHandler{}
	}
	roleMW.mx.RUnlock()
	return handlers
}

func (roleMW *middlewareReaderRequestMap) set(event string, handler ReadableRequestHandler) {
	roleMW.mx.Lock()
	if _, ok := roleMW.m[event]; ok == false {
		roleMW.m[event] = []ReadableRequestHandler{}
	}
	roleMW.m[event] = append(roleMW.m[event], handler)
	roleMW.mx.Unlock()
}

type busyMutex struct {
	mutex  sync.Mutex
	busyMX sync.RWMutex
	busy   bool
}

func (mm *busyMutex) Lock() {
	mm.busyMX.Lock()
	mm.busy = true
	mm.mutex.Lock()
	mm.busyMX.Unlock()
}

func (mm *busyMutex) Unlock() {
	mm.busyMX.Lock()
	mm.mutex.Unlock()
	mm.busy = false
	mm.busyMX.Unlock()
}

func (mm *busyMutex) isBusy() bool {
	mm.busyMX.RLock()
	busy := mm.busy
	mm.busyMX.RUnlock()
	return busy
}
