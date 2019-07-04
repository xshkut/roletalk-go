package roletalk

import (
	"sync"
)

//Role represents a service on the local Peer.
//It should handle incoming messages, requests and stream requests for certain functionality,
//e.g. "logger", "mysql_service", "auth_users" and so on
type Role struct {
	name           string
	peer           *Peer
	active         bool
	stateMutex     sync.RWMutex
	mwMessage      *middlewareMessageMap
	mwRequest      *middlewareRequestMap
	mwReader       *middlewareReaderRequestMap
	mwWriter       *middlewareWriterRequestMap
	statusHandlers []func()
}

//MessageHandler is function which handles incoming messages.
type MessageHandler func(im *MessageContext)

//RequestHandler is function which handles incoming requests.
type RequestHandler func(im *RequestContext)

//ReadableRequestHandler is function which handles incoming requests.
type ReadableRequestHandler func(im *ReaderRequestContext)

//WritableRequestHandler is function which handles incoming requests.
type WritableRequestHandler func(im *WriterRequestContext)

//ReaderHandler is function which handles incoming requests.
// type ReaderHandler func(im *RequestContext)

//WriterHandler is function which handles incoming requests.
// type WriterHandler func(im *WriterContext)

//Enable starts peer to serve the role; immediately shows the role to all connected units
func (role *Role) Enable() {
	role.stateMutex.Lock()
	prev := role.active
	role.active = true
	role.stateMutex.Unlock()
	if prev == false {
		for _, h := range role.statusHandlers {
			h()
		}
		go role.peer.broadcastRoles()
	}
}

//Disable stops peer to serve the role; immediately hides the role for all connected units
func (role *Role) Disable() {
	role.stateMutex.Lock()
	prev := role.active
	role.active = false
	role.stateMutex.Unlock()
	if prev == true {
		for _, h := range role.statusHandlers {
			h()
		}
		go role.peer.broadcastRoles()
	}
}

//Active is used to check the role's state; returns true if Peer serves the role.
func (role *Role) Active() bool {
	role.stateMutex.RLock()
	defer role.stateMutex.RUnlock()
	return role.active
}

//OnMessage registers handler for provided event. It does not support wildcard or regexp matching.
//If you need to use middleware, consider Peer.OnMessage("", handler), Peer.OnMessage(event, handler), Role.OnMessage("", handler) instead.
//Providing empty string as event sets handler for all messages of type "One way"
func (role *Role) OnMessage(event string, handler func(im *MessageContext)) {
	role.mwMessage.set(event, handler)
}

//OnRequest registers handler for provided event. It does not support wildcard or regexp matching.
//If you need to use middleware, consider Peer.OnRequest("", handler), Peer.OnRequest(event, handler), Role.OnRequest("", handler) instead.
//Providing empty string as event sets handler for all messages of type "request"
func (role *Role) OnRequest(event string, handler func(im *RequestContext)) {
	role.mwRequest.set(event, handler)
}

//OnReader ...
func (role *Role) OnReader(event string, handler func(ctx *ReaderRequestContext)) {
	role.mwReader.set(event, handler)
}

//OnWriter ...
func (role *Role) OnWriter(event string, handler func(ctx *WriterRequestContext)) {
	role.mwWriter.set(event, handler)
}

//Name returns Role's name
func (role *Role) Name() string {
	return role.name
}

//OnStatusChange registers handler for role status change (when it gets activated or deactivated)
func (role *Role) OnStatusChange(fnc func()) {
	role.stateMutex.Lock()
	role.statusHandlers = append(role.statusHandlers, fnc)
	role.stateMutex.Unlock()
}
