package roletalk

import (
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type connLocker struct {
	conn         *websocket.Conn
	mx           sync.Mutex
	locked       bool
	lockedMx     sync.RWMutex
}

func createConnLocker(conn *websocket.Conn) *connLocker {
	return &connLocker{conn: conn}
}

func (cl *connLocker) isLocked() bool {
	var locked bool
	cl.lockedMx.RLock()
	locked = cl.locked
	cl.lockedMx.RUnlock()
	return locked
}

func (cl *connLocker) Lock() {
	cl.lockedMx.Lock()
	cl.locked = true
	cl.lockedMx.Unlock()
	cl.mx.Lock()
}

func (cl *connLocker) Unlock() {
	cl.mx.Unlock()
	cl.lockedMx.Lock()
	cl.locked = false
	cl.lockedMx.Unlock()
}

func (cl *connLocker) WriteMessage(messageType int, data []byte) error {
	var err error
	cl.Lock()
	err = cl.conn.WriteMessage(messageType, data)
	cl.Unlock()
	return err
}

func (cl *connLocker) ReadMessage() (messageType int, p []byte, err error) {
	cl.Lock()
	messageType, p, err = cl.conn.ReadMessage()
	cl.Unlock()
	return
}

func (cl *connLocker) NextWriter(messageType int) (io.WriteCloser, error) {
	return cl.conn.NextWriter(messageType)
}

func (cl *connLocker) NextReader() (messageType int, r io.Reader, err error) {
	return cl.conn.NextReader()
}

func (cl *connLocker) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return cl.conn.WriteControl(messageType, data, deadline)
}

func (cl *connLocker) SetPongHandler(h func(appData string) error) {
	cl.conn.SetPongHandler(h)
}
