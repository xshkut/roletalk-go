package roletalk

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type correlation uint64

type callback struct {
	ctx *MessageContext
	err error
}

type streamCallback struct {
	err error
	res *MessageContext
}

type emitStruct struct {
	role            string
	event           string
	timeout         time.Duration
	ignoreUnitClose bool
	data            interface{}
}

func (unit *Unit) send(headers emitStruct) error {
	marked, err := markDataType(headers.data)
	if err != nil {
		return err
	}
	serialized := serializeOneway(headers.role, headers.event, marked)
	_, err = unit.writeMsgToSomeConnection(serialized)
	return err
}

func (unit *Unit) request(headers emitStruct) (*MessageContext, error) {
	marked, err := markDataType(headers.data)
	if err != nil {
		return nil, err
	}
	timeout := requestTimeout
	if headers.timeout != 0 {
		timeout = headers.timeout
	}
	corr, ch := unit.callbackCtr.prepare(timeout, headers.ignoreUnitClose)
	if _, err = unit.writeMsgToSomeConnection(serializeRequest(headers.role, headers.event, corr, marked)); err != nil {
		unit.callbackCtr.respond(corr, &callback{nil, err})
	}
	cb := <-ch
	res := cb.ctx
	if res != nil {
		res.role = headers.role
		res.event = headers.event
	}
	return res, cb.err
}

func (unit *Unit) newReader(headers emitStruct) (*MessageContext, *Readable, error) {
	var conn *connLocker
	marked, err := markDataType(headers.data)
	if err != nil {
		return nil, nil, err
	}
	timeout := requestTimeout
	if headers.timeout != 0 {
		timeout = headers.timeout
	}
	corr, ch := unit.callbackCtr.prepare(timeout, headers.ignoreUnitClose)
	channel, _ := unit.streamCtr.createStream()
	if conn, err = unit.writeMsgToSomeConnection(serializeStreamRequest(typeReader, headers.role, headers.event, corr, channel, marked)); err != nil {
		unit.callbackCtr.respond(corr, &callback{nil, err})
	}
	cb := <-ch
	ctx := cb.ctx
	if ctx != nil {
		ctx.role = headers.role
		ctx.event = headers.event
	}
	readable := &Readable{c: channel, conn: conn, pref: createStreamPrefix(channel, streamByteQuota), unit: unit, quotaSize: defQuotaSizeBytes, quotaRem: defQuotaSizeBytes}
	unit.streamCtr.bindConn(conn, channel)
	return ctx, readable, cb.err
}

func (unit *Unit) newWriter(headers emitStruct) (*MessageContext, *Writable, error) {
	var conn *connLocker
	marked, err := markDataType(headers.data)
	if err != nil {
		return nil, nil, err
	}
	timeout := requestTimeout
	if headers.timeout != 0 {
		timeout = headers.timeout
	}
	corr, ch := unit.callbackCtr.prepare(timeout, headers.ignoreUnitClose)
	channel, streamChannel := unit.streamCtr.createStream()
	if conn, err = unit.writeMsgToSomeConnection(serializeStreamRequest(typeWriter, headers.role, headers.event, corr, channel, marked)); err != nil {
		unit.callbackCtr.respond(corr, &callback{nil, err})
	}
	cb := <-ch
	ctx := cb.ctx
	if ctx != nil {
		ctx.role = headers.role
		ctx.event = headers.event
	}
	writable := &Writable{unit: unit, conn: conn, pref: createStreamPrefix(channel, streamByteChunk), streamChannel: streamChannel, quotaRem: defQuotaSizeBytes}
	unit.streamCtr.bindConn(conn, channel)
	return ctx, writable, cb.err
}

func createStreamPrefix(channel correlation, streamByte byte) []byte {
	chanBytes := serializeCorrelation(channel)
	chanLen := len(chanBytes)
	pref := []byte{typeStreamData, byte(chanLen)}
	pref = append(pref, chanBytes...)
	pref = append(pref, streamByte)
	return pref
}

func (unit *Unit) bindConn(conn *connLocker) {
	if _, loaded := unit.connections.LoadOrStore(conn, struct{}{}); loaded == true {
		return
	}
	go unit.readConnMessages(conn, unit.peer.incMsgChan)
	if os.Getenv("ENV_VAR") != "DEBUG" {
		go unit.heartBeatConn(conn)
	} else {
		// fmt.Println("Running in DEBUG mode. Connections heartbeat disabled")
	}
}

func (unit *Unit) deleteConnection(conn *connLocker, err error) {
	peer := unit.peer
	unit.streamCtr.onConnClosed(conn, err)
	unit.connections.Delete(conn)
	size := 0
	unit.connections.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	if size < 1 {
		peer.deleteUnit(unit, err)
	}
	if addr, ok := peer.addrUnits.loadByConn(conn); ok == true {
		peer.addrUnits.unbindAddr(addr)
		go peer.startReconnCycle(addr, false)
	}
}

func (unit *Unit) writeMsgToSomeConnection(msg []byte) (*connLocker, error) {
	n := 0
	var sent *connLocker
	omittedConns := make([]*connLocker, 0)

	unit.connections.Range(func(key, value interface{}) bool {
		n++
		conn, ok := key.(*connLocker)
		if ok == false {
			return true
		}
		if conn.isLocked() == true {
			omittedConns = append(omittedConns, conn)
			return true
		}
		if err := unit.writeToConn(conn, msg); err == nil {
			sent = conn
			return false
		}
		return true
	})

	if sent != nil {
		return sent, nil
	}

	rnd := time.Now().Nanosecond() / 1000
	N := len(omittedConns)
	for i := 0; i < N; i++ {
		conn := omittedConns[(i+rnd)%N]
		if _, ok := unit.connections.Load(conn); ok == false {
			break
		}
		if err := unit.writeToConn(conn, msg); err == nil {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("No available connections to send data. Tried connections: %v, unit: %v", n, unit.id)
}

func (unit *Unit) writeToConn(conn *connLocker, msg []byte) error {
	// conn.Lock()
	err := conn.WriteMessage(websocket.BinaryMessage, msg)
	// conn.Unlock()
	if err != nil {
		unit.deleteConnection(conn, err)
	}
	return err
}

func (unit *Unit) readConnMessages(conn *connLocker, ch chan<- *MessageContext) {
	var nBytesRead, chanLen int
	var way, strFlag byte
	var channel correlation
	var ok bool
	var raw, pas []byte
	var err error
	var reader io.Reader
	// var buf *bytes.Buffer
	var streamCHannel *streamChannel

	preAllocSlices := [7][]byte{make([]byte, 1), make([]byte, 2), make([]byte, 3), make([]byte, 4), make([]byte, 5), make([]byte, 6), make([]byte, 7)}
	byteContainer := make([]byte, 1)

	//handling close
	defer func() {
		if r := recover(); r != nil {
			// fmt.Println(r)
			err = fmt.Errorf("%v", r)
		} else {
			// fmt.Println("Closing conn without error")
		}
		unit.deleteConnection(conn, err)
	}()

	for {
		_, reader, err = conn.NextReader()
		if err != nil {
			panic("Error while calling conn.NextReader(): " + err.Error())
		}
		// if mt != websocket.BinaryMessage {
		// 	closeConnWithCode(conn, errWrongMessageType, errTextMessageReceived)
		// 	panic("Not binary message")
		// }

		if nBytesRead, err = reader.Read(byteContainer); err != nil || nBytesRead != 1 {
			panic(errors.New("Error while reading conn"))
		}
		//handling stream chunk without allocating new slices
		way = byteContainer[0]
		if way == typeStreamData {
			if nBytesRead, err = reader.Read(byteContainer); err != nil || nBytesRead != 1 {
				panic(errors.New("Error while reading conn"))
			}
			chanLen = int(byteContainer[0])
			pas = preAllocSlices[chanLen-1]

			if nBytesRead, err = reader.Read(pas); err != nil || nBytesRead != chanLen {
				panic(errors.New("Error while reading conn"))
			}
			channel = sliceToCorrelation(pas)

			if nBytesRead, err = reader.Read(byteContainer); err != nil || nBytesRead != 1 {
				panic(errors.New("Error while reading conn"))
			}
			strFlag = byteContainer[0]
			if streamCHannel, ok = unit.streamCtr.getStreamChannel(channel); ok != true {
				closeConnWithCode(conn, errWrongMessageType, fmt.Errorf("Cannot find channel %v", channel).Error())
				panic(fmt.Errorf("Channel not found: %v", channel))
			}
			switch strFlag {
			case streamByteChunk:
				streamCHannel.mx.Lock()
				streamCHannel.buf.ReadFrom(reader)
				streamCHannel.mx.Unlock()
				sendSignal(streamCHannel.signal)
			case streamByteFinish:
				streamCHannel.mx.Lock()
				streamCHannel.err = io.EOF
				streamCHannel.mx.Unlock()
				sendSignal(streamCHannel.signal)
			case streamByteError:
				raw, err = ioutil.ReadAll(reader)
				streamCHannel.mx.Lock()
				streamCHannel.err = errors.New(string(raw))
				streamCHannel.mx.Unlock()
				sendSignal(streamCHannel.signal)
			case streamByteQuota:
				if raw, err = ioutil.ReadAll(reader); err != nil {
					panic(errors.New("Error while reading"))
				}
				q := sliceToInt(raw)
				streamCHannel.mx.Lock()
				streamCHannel.quota = streamCHannel.quota + q
				streamCHannel.mx.Unlock()
				sendSignal(streamCHannel.signal)
			}

			continue
		}
		//handling messages
		raw, err = ioutil.ReadAll(reader)
		ch <- &MessageContext{raw: raw, unit: unit, conn: conn, w: way}
	}
}

func (unit *Unit) closeWithCode(code int, msg string) {
	unit.connections.Range(func(key, val interface{}) bool {
		conn := key.(*connLocker)
		closeConnWithCode(conn, code, msg)
		return true
	})
	for _, dest := range unit.peer.destinations {
		dest.deleteUnit(unit)
	}
}

type unitConns struct {
	m map[*websocket.Conn]*busyMutex
	l sync.RWMutex
}

func (uc *unitConns) set(c *websocket.Conn, bm *busyMutex) {
	uc.l.Lock()
	uc.m[c] = bm
	uc.l.Unlock()
}

func (uc *unitConns) get(c *websocket.Conn) *busyMutex {
	uc.l.RLock()
	bm := uc.m[c]
	uc.l.RUnlock()
	return bm
}

func (uc *unitConns) delete(c *websocket.Conn) {
	uc.l.Lock()
	delete(uc.m, c)
	uc.l.Unlock()
}

func (unit *Unit) generateAcquaintMsg(addr string) acquaintMsg {
	return acquaintMsg{ID: unit.id, Roles: unit.GetRoles(), Address: addr}
}

func (unit *Unit) heartBeatConn(conn *connLocker) {
	pong := make(chan interface{}, 1)
	conn.SetPongHandler(func(appData string) error {
		select {
		case pong <- struct{}{}:
		default:
		}
		return nil
	})
	for {
		time.Sleep(heartBeatInterval)
		if _, ok := unit.connections.Load(conn); ok == false {
			return
		}
		timeout := time.Now().Add(heartBeatTimeout)
		if conn.WriteControl(websocket.PingMessage, []byte{}, timeout) != nil {
			return
		}
		select {
		case <-pong:
		case <-time.After(heartBeatTimeout):
			closeConnWithCode(conn, errHeartbeatTimeout, "heartbeat timeout")
		}
	}
}

type reqCallbackController struct {
	m  map[correlation]cbWaiter
	mx *sync.RWMutex
	ch chan correlation
}

type cbWaiter struct {
	ch           chan *callback
	timer        *time.Timer
	ignUnitClose bool
}

func produceCorrelation(cb interface{}, corrChan chan<- correlation, callbacksMx *sync.RWMutex) {
	callbacks, _ := cb.(map[correlation]interface{})
	var i correlation
	var ok bool
	for {
		i++
		if i > maxCorrelation {
			i = 0
		}
		if _, ok = callbacks[i]; ok == true {
			break
		}
		corrChan <- i
	}
}

func rangeConnForCloseFunc(key, val interface{}) bool {
	if conn, ok := key.(*connLocker); ok == true {
		conn := conn
		go func() {
			closeConnWithCode(conn, errManualClose, "auth rejected")
		}()
	} else {
		panic(fmt.Errorf("Unit.connection's key is not *websocket.Conn"))
	}
	return true
}

func createCallbackController() reqCallbackController {
	rcm := reqCallbackController{}
	rcm.mx = new(sync.RWMutex)
	rcm.m = make(map[correlation]cbWaiter)
	rcm.ch = make(chan correlation, runtime.NumCPU()*2)
	go produceCorrelation(rcm.m, rcm.ch, rcm.mx)
	return rcm
}

func (rcm *reqCallbackController) prepare(timeout time.Duration, ignUnitClose bool) (corr correlation, ch chan *callback) {
	ch = make(chan *callback, 1)
	corr = <-rcm.ch
	timer := time.AfterFunc(timeout, func() {
		rcm.respond(corr, &callback{err: fmt.Errorf("Request timeout: %v", timeout)})
	})
	rcm.mx.Lock()
	rcm.m[corr] = cbWaiter{
		ch,
		timer,
		ignUnitClose,
	}
	rcm.mx.Unlock()
	return
}

func (rcm *reqCallbackController) respond(corr correlation, cb *callback) {
	rcm.mx.Lock()
	defer rcm.mx.Unlock()
	cw, ok := rcm.m[corr]
	if ok == false {
		return
	}
	cw.ch <- cb
	close(cw.ch)
	cw.timer.Stop()
	delete(rcm.m, corr)
}

func (rcm *reqCallbackController) onClose() {
	cb := &callback{err: errors.New("Unit closed")}
	rcm.mx.Lock()
	defer rcm.mx.Unlock()
	for corr, cw := range rcm.m {
		if cw.ignUnitClose == true {
			break
		}
		cw.ch <- cb
		close(cw.ch)
		cw.timer.Stop()
		delete(rcm.m, corr)
	}
}

func (unit *Unit) runOnClose(err error) {
	for _, handler := range unit.closeHandlers {
		handler(err)
	}
}

func sendSignal(ch chan<- interface{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (unit *Unit) setLastRoleSession(i int) {
	unit.rolesMx.Lock()
	unit.lastRoleSession = i
	unit.rolesMx.Unlock()
}

func (unit *Unit) getLastRoleSession() int {
	unit.rolesMx.RLock()
	i := unit.lastRoleSession
	unit.rolesMx.RUnlock()
	return i
}
