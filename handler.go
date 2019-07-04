package roletalk

import (
	"encoding/json"
	"errors"
	"fmt"
)

func (peer *Peer) consumeIncomingMessages() {
	for inc := range peer.incMsgChan {
		peer.serveIncMsg(inc)
	}
}

func (peer *Peer) serveIncMsg(ctx *MessageContext) {
	defer func() {
		if r := recover(); r != nil {
			go ctx.unit.closeWithCode(errIncorrectMessageStructure, fmt.Sprint(r))
		}
	}()
	var role *Role
	var hasRole bool
	var err error
	switch ctx.w {
	case typeMessage:
		roleName, event, t, rawData := parseOneway(ctx.raw)
		ctx.role = roleName
		ctx.origin.T = t
		ctx.origin.Data = rawData
		ctx.event = event
		ctx.Data, err = retrieveDataByType(t, rawData)
		if err != nil {
			go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong data type: %v", t))
			return
		}
		if role, hasRole = peer.getRole(roleName); hasRole == false {
			return
		}
		go role.emitMsg(ctx)
	case typeRequest:
		ctx := &RequestContext{MessageContext: ctx}
		// ctx := &RequestContext{Conn: ctx.conn, Unit: ctx.unit, Event: ctx.event, Role: ctx.role, Data: ctx.Data, origin: ctx.origin, w: ctx.w, raw: ctx.raw}
		roleName, event, corr, t, rawData := parseRequest(ctx.raw)
		ctx.role = roleName
		ctx.origin.T = t
		ctx.origin.Data = rawData
		ctx.event = event
		ctx.corr = corr
		ctx.Data, err = retrieveDataByType(t, rawData)
		if err != nil {
			go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong data type: %v", t))
			return
		}
		if role, hasRole = peer.getRole(roleName); hasRole == false {
			ctx.Reject(fmt.Sprintf("No such role [%v] on peer %v", roleName, peer.id))
			return
		}
		go role.emitRequest(ctx)
	case typeReader:
		rc := &RequestContext{MessageContext: ctx}
		ctx := &WriterRequestContext{RequestContext: rc}
		// ctx := &RequestContext{Conn: ctx.conn, Unit: ctx.unit, Event: ctx.event, Role: ctx.role, Data: ctx.Data, origin: ctx.origin, w: ctx.w, raw: ctx.raw}
		roleName, event, corr, channel, t, rawData := parseStreamRequest(ctx.raw)
		ctx.role = roleName
		ctx.event = event
		ctx.origin.T = t
		ctx.origin.Data = rawData
		ctx.corr = corr
		ctx.channel = channel
		ctx.Data, err = retrieveDataByType(t, rawData)
		if err != nil {
			go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong data type: %v", t))
			return
		}
		if role, hasRole = peer.getRole(roleName); hasRole == false {
			ctx.Reject(fmt.Sprintf("No such role [%v] on peer %v", roleName, peer.id))
			return
		}
		go role.emitWriter(ctx)
	case typeWriter:
		rc := &RequestContext{MessageContext: ctx}
		ctx := &ReaderRequestContext{RequestContext: rc}
		// ctx := &RequestContext{conn: ctx.conn, Unit: ctx.unit, Event: ctx.event, Role: ctx.role, Data: ctx.Data, origin: ctx.origin, w: ctx.w, raw: ctx.raw}
		roleName, event, corr, channel, t, rawData := parseStreamRequest(ctx.raw)
		ctx.role = roleName
		ctx.origin.T = t
		ctx.origin.Data = rawData
		ctx.event = event
		ctx.corr = corr
		ctx.channel = channel
		ctx.Data, err = retrieveDataByType(t, rawData)
		if err != nil {
			go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong data type: %v", t))
			return
		}
		if role, hasRole = peer.getRole(roleName); hasRole == false {
			ctx.Reject(fmt.Sprintf("No such role [%v] on peer %v", roleName, peer.id))
			return
		}
		go role.emitReader(ctx)
	case typeStreamResolve:
		// ctx := &StreamReponseContext{MessageContext: ctx}
		corr, channel, t, rawData := parseStreamResponse(ctx.raw)
		ctx.channel = channel
		ctx.origin.T = t
		ctx.origin.Data = rawData
		ctx.Data, err = retrieveDataByType(t, rawData)
		if err != nil {
			go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong data type: %v", t))
			return
		}
		go ctx.unit.callbackCtr.respond(corr, &callback{ctx: ctx})
	case typeStreamReject:
		// ctx := StreamReponseContext{MessageContext: ctx}
		corr, channel, t, rawData := parseStreamResponse(ctx.raw)
		ctx.channel = channel
		ctx.origin.T = t
		ctx.origin.Data = rawData
		ctx.Data, err = retrieveDataByType(t, rawData)
		if err != nil {
			go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong data type: %v", t))
			return
		}
		go ctx.unit.callbackCtr.respond(corr, &callback{ctx: ctx, err: errors.New(string(ctx.raw))})
	case typeResolve:
		corr, t, rawData := parseResponse(ctx.raw)
		ctx.origin.T = t
		ctx.origin.Data = rawData
		ctx.Data, err = retrieveDataByType(t, rawData)
		if err != nil {
			go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong data type: %v", t))
			return
		}
		ctx.unit.callbackCtr.respond(corr, &callback{ctx: ctx})
	case typeReject:
		corr, t, rawData := parseResponse(ctx.raw)
		ctx.origin.T = t
		ctx.origin.Data = rawData
		ctx.Data, err = retrieveDataByType(t, rawData)
		if err != nil {
			go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong data type: %v", t))
			return
		}
		go ctx.unit.callbackCtr.respond(corr, &callback{err: errors.New(string(rawData)), ctx: ctx})
	case typeRoles:
		roles, err := parseRoles(ctx.raw)
		if err != nil {
			go closeConnWithCode(ctx.conn, errIncorrectMessageStructure, fmt.Sprintf("wrong roles message message: %v", string(ctx.raw)))
			return
		}
		newRoles := make(map[string]interface{})
		for _, role := range roles {
			newRoles[role] = struct{}{}
		}
		ctx.unit.rolesMx.Lock()
		ctx.unit.roles = newRoles
		ctx.unit.rolesMx.Unlock()
		go peer.onNewUnitRoles(ctx.unit)
	case typeAcquaint:
		am := acquaintMsg{}
		if err := json.Unmarshal(ctx.raw, &am); err != nil {
			go closeConnWithCode(ctx.conn, errIncorrectMessageStructure, fmt.Sprintf("wrong acquaint message structure: %v", string(ctx.raw)))
		}
		if peer.Friendly == false || peer.Unit(am.ID) != nil {
			return
		}
		for _, dest := range peer.ListDestinations() {
			for _, role := range am.Roles {
				if role == dest {
					go peer.Connect(am.Address, ConnectOptions{})
					return
				}
			}
		}
	default:
		go closeConnWithCode(ctx.conn, errWrongMessageType, fmt.Sprintf("wrong message type: %v", ctx.raw[0]))
	}
}
