package roletalk

import "fmt"

func (peer *Peer) createRole(name string) *Role {
	return &Role{
		name:      name,
		peer:      peer,
		active:    true,
		mwMessage: createMiddlewareMessageMap(),
		mwRequest: createMiddlewareRequestMap(),
		mwReader:  createMiddlewareReaderMap(),
		mwWriter:  createMiddlewareWriterMap(),
	}
}

func (role *Role) emitRequest(ctx *RequestContext) {
	mwChain := role.mwRequest
	for _, mw := range mwChain.get("") {
		if ctx.r == true {
			break
		}
		mw(ctx)
	}
	for _, mw := range mwChain.get(ctx.event) {
		if ctx.r == true {
			break
		}
		mw(ctx)
	}
	if ctx.r == false {
		ctx.runCallbacks()
		if ctx.Err != nil {
			ctx.Reject(ctx.Err)
		} else if ctx.Res != nil {
			ctx.Reply(ctx.Res)
		} else {
			ctx.Reject(fmt.Sprintf("Event [%v] is not handled by the peer [%v]", ctx.event, role.peer.id))
		}
	}
}

func (role *Role) emitReader(ctx *ReaderRequestContext) {
	mwChain := role.mwReader
	for _, mw := range mwChain.get("") {
		if ctx.r == true {
			break
		}
		mw(ctx)
	}
	for _, mw := range mwChain.get(ctx.event) {
		if ctx.r == true {
			break
		}
		mw(ctx)
	}
	if ctx.r == false {
		ctx.runCallbacks()
		if ctx.Err != nil {
			ctx.Reject(ctx.Err)
		} else if ctx.Res != nil {
			ctx.Reply(ctx.Res)
		} else {
			ctx.Reject(fmt.Sprintf("Event [%v] is not handled by the peer [%v]", ctx.event, role.peer.id))
		}
	}
}

func (role *Role) emitWriter(ctx *WriterRequestContext) {
	mwChain := role.mwWriter
	for _, mw := range mwChain.get("") {
		if ctx.r == true {
			break
		}
		mw(ctx)
	}
	for _, mw := range mwChain.get(ctx.event) {
		if ctx.r == true {
			break
		}
		mw(ctx)
	}
	if ctx.r == false {
		ctx.runCallbacks()
		if ctx.Err != nil {
			ctx.Reject(ctx.Err)
		} else if ctx.Res != nil {
			ctx.Reply(ctx.Res)
		} else {
			ctx.Reject(fmt.Sprintf("Event [%v] is not handled by the peer [%v]", ctx.event, role.peer.id))
		}
	}
}

func (role *Role) emitMsg(im *MessageContext) {
	mwChain := role.mwMessage
	for _, mw := range mwChain.get("") {
		mw(im)
	}
	for _, mw := range mwChain.get(im.event) {
		mw(im)
	}
}
