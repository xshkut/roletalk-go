package roletalk

import (
	"errors"
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

//OriginData represents unchanged received context's data.
type OriginData struct {
	T    Datatype
	Data []byte
}

//MessageContext is context for one-way message and responses, including ones for readable and writable streams.
//T and Data are allowed to be changed by middleware. To get original data call OriginData().
type MessageContext struct {
	conn    *connLocker
	unit    *Unit
	role    string
	event   string
	Data    interface{}
	origin  OriginData
	w       byte
	raw     []byte
	channel correlation //specific for streams
}

//Role ...
func (ctx *MessageContext) Role() string {
	return ctx.role
}

//Event ...
func (ctx *MessageContext) Event() string {
	return ctx.event
}

//Unit ...
func (ctx *MessageContext) Unit() *Unit {
	return ctx.unit
}

//Conn ...
func (ctx *MessageContext) Conn() *websocket.Conn {
	return ctx.conn.conn
}

func (ctx *MessageContext) String() string {
	return string(ctx.origin.Data)
}

//OriginData returns unchanged received context's data.
func (ctx *MessageContext) OriginData() OriginData {
	return ctx.origin
}

//RequestContext ...
//Implements Reply(), Reject()
type RequestContext struct {
	*MessageContext
	//specific for Request
	Res  interface{}
	Err  error
	corr correlation
	r    bool
	cbs  []RequestHandler
}

//Reply stops middleware flow and responds to the message. If data argument is provided, it overwrites im.Data
func (ctx *RequestContext) Reply(data interface{}) error {
	var t byte
	var d interface{}
	var res []byte
	ctx.r = true
	tRej := typeReject
	tRes := typeResolve
	if data != nil {
		ctx.Res = data
	}

	ctx.runCallbacks()

	if ctx.Err != nil {
		d = ctx.Err
		t = tRej
	} else {
		d = ctx.Res
		t = tRes
	}
	b, e := markDataType(d)
	if e != nil {
		return e
	}

	res = serializeResponse(t, ctx.corr, b)
	_, err := ctx.Unit().writeMsgToSomeConnection(res)

	return err
}

//Reject responds to request with error; data can be error, string or nil
func (ctx *RequestContext) Reject(data interface{}) error {
	ctx.r = true
	switch d := data.(type) {
	case error:
		ctx.Err = d
	case string:
		ctx.Err = errors.New(d)
	case nil:
	default:
		return errors.New("data argument should be error, string or nil")
	}
	ctx.runCallbacks()
	b, err := markDataType(ctx.Err)
	if err != nil {
		return err
	}
	_, err = ctx.Unit().writeMsgToSomeConnection(serializeResponse(typeReject, ctx.corr, b))
	return err
}

//OriginData returns unchanged received context's data.
func (ctx *RequestContext) OriginData() OriginData {
	return ctx.origin
}

//Then binds middleware to message context. All middleware runs in LIFO order
func (ctx *RequestContext) Then(cb func(ctx *RequestContext)) {
	ctx.cbs = append(ctx.cbs, cb)
}

func (ctx *RequestContext) runCallbacks() {
	len := len(ctx.cbs)
	for i := len - 1; i >= 0; i-- {
		fnc := ctx.cbs[i]
		fnc(ctx)
	}
}

//ReaderRequestContext ...
type ReaderRequestContext struct {
	*RequestContext
	cbs []ReadableRequestHandler
}

//Reply stops middleware flow and responds to the message. If data argument is provided, it overwrites im.Data
func (ctx *ReaderRequestContext) Reply(data interface{}) (*Readable, error) {
	var t byte
	var d interface{}
	var res []byte
	var channel correlation
	// var sc *streamChannel

	ctx.r = true

	if data != nil {
		ctx.Res = data
	}

	ctx.runCallbacks()

	if ctx.Err != nil {
		d = ctx.Err
		t = typeStreamReject
	} else {
		d = ctx.Res
		t = typeStreamResolve
	}
	b, e := markDataType(d)
	if e != nil {
		return nil, e
	}

	channel, _ = ctx.Unit().streamCtr.createStream()
	res = serializeStreamResponse(t, ctx.corr, channel, b)
	conn, err := ctx.Unit().writeMsgToSomeConnection(res)
	if err != nil {
		ctx.Unit().streamCtr.delete(channel)
		return nil, err
	}
	readable := &Readable{
		unit:      ctx.Unit(),
		conn:      conn,
		c:         channel,
		pref:      createStreamPrefix(channel, streamByteQuota),
		quotaRem:  defQuotaSizeBytes,
		quotaSize: defQuotaSizeBytes,
	}
	ctx.unit.streamCtr.bindConn(conn, channel)
	return readable, nil
}

//Then binds middleware to message context. All middleware runs in LIFO order
func (ctx *ReaderRequestContext) Then(cb func(ctx *ReaderRequestContext)) {
	ctx.cbs = append(ctx.cbs, cb)
}

//WriterRequestContext ...
type WriterRequestContext struct {
	*RequestContext
	cbs []WritableRequestHandler
}

//Then binds middleware to message context. All middleware runs in LIFO order
func (ctx *WriterRequestContext) Then(cb func(ctx *WriterRequestContext)) {
	ctx.cbs = append(ctx.cbs, cb)
}

//Reply stops middleware flow and responds to the message. If data argument is provided, it overwrites im.Data
func (ctx *WriterRequestContext) Reply(data interface{}) (*Writable, error) {
	var t byte
	var d interface{}
	var res []byte
	var channel correlation
	var sc *streamChannel

	ctx.r = true

	if data != nil {
		ctx.Res = data
	}

	ctx.runCallbacks()

	if ctx.Err != nil {
		d = ctx.Err
		t = typeStreamReject
	} else {
		d = ctx.Res
		t = typeStreamResolve
	}
	b, e := markDataType(d)
	if e != nil {
		return nil, e
	}

	channel, sc = ctx.Unit().streamCtr.createStream()
	res = serializeStreamResponse(t, ctx.corr, channel, b)
	conn, err := ctx.Unit().writeMsgToSomeConnection(res)
	if err != nil {
		ctx.Unit().streamCtr.delete(channel)
		return nil, err
	}
	writable := &Writable{unit: ctx.Unit(), conn: conn, pref: createStreamPrefix(channel, streamByteChunk), streamChannel: sc, quotaRem: defQuotaSizeBytes}
	ctx.unit.streamCtr.bindConn(conn, channel)
	return writable, nil
}

//Readable implements Reader
type Readable struct {
	unit      *Unit
	conn      *connLocker
	c         correlation
	pref      []byte
	quotaRem  int
	quotaSize int
	quotaMx   sync.RWMutex
}

func (r *Readable) Read(p []byte) (n int, err error) {
	unit := r.unit
	streamCtr := &unit.streamCtr
	c := r.c
	sc, ok := streamCtr.getStreamChannel(c)
	if ok == false {
		return 0, errors.New("Stream closed")
	}
	buf := &sc.buf
	sc.mx.Lock()
	n, err = buf.Read(p)
	scErr := sc.err
	sc.mx.Unlock()
	switch {
	case n != 0:
		q := r.addRemQuota(-n)
		if float64(q) < float64(r.quotaSize)*defQuotaThreshold {
			go r.addQuota(r.quotaSize - q)
		}
		return n, nil
	case scErr != nil:
		return 0, scErr
	case err != io.EOF:
		return 0, err
	default:
		<-sc.signal
		return r.Read(p)
	}
}

func (r *Readable) addQuota(q int) (err error) {
	var writer io.WriteCloser
	r.addRemQuota(q)
	quotaSlice := serializeInt(q)
	r.conn.Lock()
	writer, err = r.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		r.conn.Unlock()
		return err
	}
	if _, err = writer.Write(r.pref); err != nil {
		r.conn.Unlock()
		return err
	}
	if _, err = writer.Write(quotaSlice); err != nil {
		r.conn.Unlock()
		return err
	}
	writer.Close()
	r.conn.Unlock()
	return
}

// func (r *Readable) setWaterMark(n int) {
// 	if n <= 0 {
// 		panic(errors.New(errNonPositiveWaterMark))
// 	}
// 	r.quotaSize = n
// }

//Destroy sends err to writable end and closes stream
func (r *Readable) Destroy(err error) error {
	errMsg := append(r.pref[0:len(r.pref)-1], streamByteError)
	errMsg = append(errMsg, []byte(err.Error())...)
	return r.unit.writeToConn(r.conn, errMsg)
}

func (r *Readable) addRemQuota(n int) int {
	r.quotaMx.Lock()
	q := r.quotaRem + n
	r.quotaRem = q
	r.quotaMx.Unlock()
	return q
}

func (r *Readable) getRemQuota(n int) int {
	r.quotaMx.RLock()
	q := r.quotaRem
	r.quotaMx.RUnlock()
	return q
}

//Writable implement WriteCLoser
type Writable struct {
	unit *Unit
	conn *connLocker
	// c             correlation
	pref          []byte
	streamChannel *streamChannel
	quotaRem      int
}

func (w *Writable) Write(p []byte) (n int, err error) {
	var writer io.WriteCloser
	var quota int
	err = w.streamChannel.getErr()
	if err != nil {
		return 0, w.streamChannel.err
	}
	w.streamChannel.mx.Lock()
	quota = w.streamChannel.quota
	w.streamChannel.mx.Unlock()
	w.quotaRem += quota
	select {
	case <-w.streamChannel.signal:
		if err = w.streamChannel.err; err != nil {
			return 0, err
		}
	default:
	}
	if w.quotaRem <= 0 {
		<-w.streamChannel.signal
		return w.Write(p)
	}

	w.conn.Lock()
	writer, err = w.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		w.conn.Unlock()
		return 0, err
	}
	if _, err = writer.Write(w.pref); err != nil {
		w.conn.Unlock()
		return 0, err
	}
	if n, err = writer.Write(p); err != nil {
		w.conn.Unlock()
		return 0, err
	}
	err = writer.Close()
	w.conn.Unlock()
	w.quotaRem -= n
	return
}

//Close successfully
func (w *Writable) Close() error {
	errMsg := append(w.pref[0:len(w.pref)-1], streamByteFinish)
	err := w.unit.writeToConn(w.conn, errMsg)
	return err
}

//Destroy sends err to readable end and closes stream
func (w *Writable) Destroy(err error) error {
	errMsg := append(w.pref[0:len(w.pref)-1], streamByteError)
	errMsg = append(errMsg, []byte(err.Error())...)
	return w.unit.writeToConn(w.conn, errMsg)
}
