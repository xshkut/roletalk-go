package roletalk

import (
	"crypto"
	"crypto/rand"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"gotest.tools/assert"
)

var peerOne = NewPeer(PeerOptions{Name: "peer one"})
var peerTwo = NewPeer(PeerOptions{Name: "peer two"})
var address string

func TestCommunication(t *testing.T) {
	peerOne.Role("echo")
	peerTwo.Destination("echo")
	addr, err := peerOne.Listen("localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	address = addr.String()
	_, err = peerTwo.Connect("ws://"+address, ConnectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 10)
	t.Run("Testing one-way message", testMessage)
	t.Run("Testing request", testRequest)
	t.Run("Testing request timeout", testRequestTimeout)
	t.Run("Testing reader", testReader)
	t.Run("Testing writer", testWriter)
	t.Run("Testing reader destroy", testReaderDestroy)
	t.Run("Testing writer destroy", testWriterDestroy)
	t.Run("Testing writer conn abort", testReaderConnAbort)
	time.Sleep(time.Millisecond * 10) //for reconnect after conn abort
	t.Run("Testing writer conn abort", testWriterConnAbort)
}

func testMessage(t *testing.T) {
	ch := make(chan interface{})
	peerOne.Role("echo").OnMessage("test", func(im *MessageContext) {
		ch <- im.Data
	})
	destEcho := peerTwo.Destination("echo")
	err := destEcho.Send("test", EmitOptions{Data: true})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-time.After(time.Second * 1):
		t.Fatal("Message timeout (1 sec)")
	case data := <-ch:
		assert.Equal(t, data, true)
	}
}

func testRequest(t *testing.T) {
	peerOne.Role("echo").OnRequest("", func(im *RequestContext) {
		if d, ok := im.Res.(string); ok == false {
			im.Res = "1"
		} else {
			im.Res = d + "1"
		}
		im.Then(func(im *RequestContext) {
			if d, ok := im.Res.(string); ok == false {
				im.Res = "1"
			} else {
				im.Res = d + "1"
			}
		})
	})
	peerOne.Role("echo").OnRequest("test", func(im *RequestContext) {
		if d, ok := im.Res.(string); ok == false {
			im.Res = "2"
		} else {
			im.Res = d + "2"
		}
		im.Then(func(im *RequestContext) {
			if d, ok := im.Res.(string); ok == false {
				im.Res = "2"
			} else {
				im.Res = d + "2"
			}
		})
		im.Reply(nil)
	})
	destEcho := peerTwo.Destination("echo")
	res, err := destEcho.Request("test", EmitOptions{Data: true})
	assert.Equal(t, err, nil)
	assert.Equal(t, res.Data, "1221")
}

func testRequestTimeout(t *testing.T) {
	peerOne.Role("echo").OnRequest("test1", func(im *RequestContext) {
		time.Sleep(time.Millisecond * 10)
		im.Reply(nil)
	})
	destEcho := peerTwo.Destination("echo")
	_, err := destEcho.Request("test1", EmitOptions{Data: true, Timeout: time.Nanosecond * 1})
	assert.Error(t, err, "Request timeout: 1ns")
}

func testReader(t *testing.T) {
	var reader io.Reader
	var writer io.WriteCloser
	var err error
	readableTested := make(chan interface{})
	wg := sync.WaitGroup{}
	receiverHash := crypto.SHA256.New()
	emitterHash := crypto.SHA256.New()

	peerOne.Role("echo").OnReader("echo", func(ctx *ReaderRequestContext) {
		reader, err = ctx.Reply(true)
		assert.Equal(t, err, nil)
		readableTested <- struct{}{}
	})

	destEcho := peerTwo.Destination("echo")
	ctx, writer, err1 := destEcho.NewWriter("echo", EmitOptions{})
	assert.NilError(t, err1)
	assert.Equal(t, ctx.Data, true)

	<-readableTested

	//READER
	wg.Add(1)
	go func() {
		sl := make([]byte, 10)
		for {
			n, err := reader.Read(sl)
			if err != nil {
				assert.Equal(t, err, io.EOF)
				break
			}
			receiverHash.Write(sl[0:n])
		}
		wg.Done()
	}()

	//WRITER
	wg.Add(1)
	go func() {
		sl := make([]byte, 1024)
		for i := 0; i < 1024; i++ {
			_, err := rand.Read(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
			_, err = writer.Write(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
			emitterHash.Write(sl)
		}
		writer.Close()
		wg.Done()
	}()

	wg.Wait()
	assert.Assert(t, reflect.DeepEqual(receiverHash.Sum(nil), emitterHash.Sum(nil)))
}

func testWriter(t *testing.T) {
	var reader io.Reader
	var writer io.WriteCloser
	var err error
	writableTested := make(chan interface{})
	wg := sync.WaitGroup{}
	receiverHash := crypto.SHA256.New()
	emitterHash := crypto.SHA256.New()

	peerOne.Role("echo").OnWriter("echo", func(ctx *WriterRequestContext) {
		writer, err = ctx.Reply(true)
		assert.Equal(t, err, nil)
		writableTested <- struct{}{}
	})

	destEcho := peerTwo.Destination("echo")
	ctx, reader, err1 := destEcho.NewReader("echo", EmitOptions{})
	assert.NilError(t, err1)
	assert.Equal(t, ctx.Data, true)

	<-writableTested

	//READER
	wg.Add(1)
	go func() {
		sl := make([]byte, 10)
		for {
			n, err := reader.Read(sl)
			if err != nil {
				assert.Equal(t, err, io.EOF)
				break
			}
			receiverHash.Write(sl[0:n])
		}
		wg.Done()
	}()

	//WRITER
	wg.Add(1)
	go func() {
		sl := make([]byte, 1024)
		for i := 0; i < 1024; i++ {
			_, err := rand.Read(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
			_, err = writer.Write(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
			emitterHash.Write(sl)
		}
		writer.Close()
		wg.Done()
	}()

	wg.Wait()
	assert.Assert(t, reflect.DeepEqual(receiverHash.Sum(nil), emitterHash.Sum(nil)))
}

func testWriterDestroy(t *testing.T) {
	var reader *Readable
	var writer *Writable
	wg := sync.WaitGroup{}
	readerCreated := make(chan interface{})
	errMsg := "some error"

	peerOne.Role("echo").OnWriter("echo1", func(ctx *WriterRequestContext) {
		writer, _ = ctx.Reply(true)
		readerCreated <- struct{}{}
	})

	destEcho := peerTwo.Destination("echo")
	ctx, reader, err := destEcho.NewReader("echo1", EmitOptions{})
	assert.NilError(t, err)
	assert.Equal(t, ctx.Data, true)

	//READER
	<-readerCreated

	wg.Add(1)
	go func() {
		sl := make([]byte, 10)
		for {
			_, err := reader.Read(sl)
			if err != nil {
				assert.Equal(t, err.Error(), errMsg)
				break
			}
		}
		wg.Done()
	}()

	//WRITER
	go func() {
		sl := make([]byte, 1)
		for i := 0; i < 1; i++ {
			_, err := rand.Read(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
			_, err = writer.Write(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
		}
		writer.Destroy(errors.New(errMsg))
	}()

	wg.Wait()
}

func testReaderDestroy(t *testing.T) {
	var reader *Readable
	var writer *Writable
	wg := sync.WaitGroup{}
	readerCreated := make(chan interface{})
	errMsg := "some error"

	peerOne.Role("echo").OnReader("echo2", func(ctx *ReaderRequestContext) {
		reader, _ = ctx.Reply(true)
		readerCreated <- struct{}{}
	})

	destEcho := peerTwo.Destination("echo")
	ctx, writer, err := destEcho.NewWriter("echo2", EmitOptions{})
	assert.NilError(t, err)
	assert.Equal(t, ctx.Data, true)

	//READER
	<-readerCreated
	wg.Add(1)
	go func() {
		sl := make([]byte, 10)
		for {
			_, err := reader.Read(sl)
			if err != nil {
				assert.Equal(t, err.Error(), errMsg)
				break
			}
		}
		wg.Done()
	}()

	//WRITER
	go func() {
		sl := make([]byte, 1)
		for i := 0; i < 1; i++ {
			_, err := rand.Read(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
			_, err = writer.Write(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
		}
		writer.Destroy(errors.New(errMsg))
	}()

	wg.Wait()
}

func testReaderConnAbort(t *testing.T) {
	var reader *Readable
	var writer *Writable
	wg := sync.WaitGroup{}
	readerCreated := make(chan interface{})

	peerOne.Role("echo").OnReader("echo3", func(ctx *ReaderRequestContext) {
		reader, _ = ctx.Reply(true)
		readerCreated <- struct{}{}
	})

	destEcho := peerTwo.Destination("echo")
	ctx, writer, err := destEcho.NewWriter("echo3", EmitOptions{})
	assert.NilError(t, err)
	assert.Equal(t, ctx.Data, true)

	//READER
	<-readerCreated
	wg.Add(1)
	go func() {
		sl := make([]byte, 10)
		defer wg.Done()
		for {
			_, err := reader.Read(sl)
			if err != nil {
				assert.ErrorContains(t, err, "1006")
				break
			}
		}
	}()

	//WRITER
	go func() {
		sl := make([]byte, 1)
		for i := 0; i < 1; i++ {
			_, err := rand.Read(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
			_, err = writer.Write(sl)
			if err != nil {
				t.Fatal(err)
				break
			}
		}
		writer.conn.conn.Close()
	}()

	wg.Wait()
}

func testWriterConnAbort(t *testing.T) {
	var reader *Readable
	var writer *Writable
	wg := sync.WaitGroup{}
	readerCreated := make(chan interface{})

	peerOne.Role("echo").OnReader("echo4", func(ctx *ReaderRequestContext) {
		reader, _ = ctx.Reply(true)
		readerCreated <- struct{}{}
	})

	destEcho := peerTwo.Destination("echo")
	ctx, writer, err := destEcho.NewWriter("echo4", EmitOptions{})
	assert.NilError(t, err)
	assert.Equal(t, ctx.Data, true)

	//READER
	<-readerCreated
	wg.Add(1)
	go func() {
		defer wg.Done()
		sl := make([]byte, 10)
		for i := 0; i < 10; i++ {
			_, err := reader.Read(sl)
			if err != nil {
				break
			}
		}
		writer.conn.conn.Close()
	}()

	//WRITER
	wg.Add(1)
	go func() {
		defer wg.Done()
		sl := make([]byte, 1)
		var err error
		for {
			_, err = rand.Read(sl)
			if err != nil {
				break
			}
			_, err = writer.Write(sl)
			if err != nil {
				break
			}
		}
		assert.ErrorContains(t, err, "closed")
	}()

	wg.Wait()
}
