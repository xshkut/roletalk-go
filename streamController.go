package roletalk

import (
	"bytes"
	"sync"

	"github.com/pkg/errors"
)

type streamController struct {
	m         map[correlation]*streamChannel
	mx        *sync.RWMutex
	ch        chan correlation
	connChans sync.Map //key: *connLocker, value: sync.Map<channel,interface{}>
}

type streamChannel struct {
	buf    bytes.Buffer
	err    error
	signal chan interface{}
	quota  int
	mx     sync.Mutex
}

func createStreamController() *streamController {
	sm := streamController{}
	sm.m = make(map[correlation]*streamChannel)
	sm.mx = new(sync.RWMutex)
	sm.ch = make(chan correlation)
	go produceCorrelation(sm.m, sm.ch, sm.mx)
	return &sm
}

func (sm *streamController) createStream() (channel correlation, sc *streamChannel) {
	channel = <-sm.ch
	sc = new(streamChannel)
	sc.signal = make(chan interface{})
	sm.mx.Lock()
	sm.m[channel] = sc
	sm.mx.Unlock()
	return channel, sc
}

func (sm *streamController) delete(channel correlation) {
	sm.mx.Lock()
	delete(sm.m, channel)
	sm.mx.Unlock()
}

func (sm *streamController) setErr(channel correlation, err error) {
	sm.mx.RLock()
	sc, ok := sm.m[channel]
	sm.mx.RUnlock()
	if ok == true {
		sc.setErr(err)
		sendSignal(sc.signal)
	}
}

func (sm *streamController) getStreamChannel(channel correlation) (*streamChannel, bool) {
	sm.mx.RLock()
	sc, ok := sm.m[channel]
	sm.mx.RUnlock()
	return sc, ok
}

func (sm *streamController) bindConn(conn *connLocker, channel correlation) {
	var c *sync.Map
	var val interface{}
	// var ok bool
	val, _ = sm.connChans.Load(conn)
	if val == nil {
		c = &sync.Map{}
		sm.connChans.Store(conn, c)
	} else {
		c = val.(*sync.Map)
	}
	c.Store(channel, struct{}{})
}

func (sm *streamController) unbindConn(conn *connLocker, channel correlation) {
	var c *sync.Map
	var val interface{}
	var ok bool
	val, ok = sm.connChans.Load(conn)
	if ok == false {
		return
	}
	c = val.(*sync.Map)
	c.Delete(channel)
}

func (sm *streamController) onConnClosed(conn *connLocker, err error) {
	key, ok := sm.connChans.Load(conn)
	if ok == false {
		return
	}
	sm.connChans.Delete(conn)
	m := key.(*sync.Map)
	m.Range(func(key, val interface{}) bool {
		c := key.(correlation)
		sm.setErr(c, errors.Wrap(err, "underlying connection closed due to error previously occured"))
		return true
	})
}

func (sc *streamChannel) setErr(err error) {
	sc.mx.Lock()
	sc.err = err
	sc.mx.Unlock()
}

func (sc *streamChannel) getErr() error {
	sc.mx.Lock()
	err := sc.err
	sc.mx.Unlock()
	return err
}
