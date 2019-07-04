package roletalk

import (
	"fmt"
	"testing"
)

const N = 1000 * 1000

func BenchmarkAllocatingChannels(b *testing.B) {
	var ch chan interface{}
	for i := 0; i < N; i++ {
		ch = make(chan interface{}, 1)
		// ch <- struct{}{}
	}
	_ = ch
}

type person struct {
	name     string
	surename string
}

func getFullName(p1 interface{}) string {
	p, _ := p1.(*woman)
	return fmt.Sprintf("%v %v", p.name, p.surename)
}

type woman struct {
	name     string
	surename string
	prev     string
}

type first struct {
	s second
}

type second struct {
	a bool
}

// func Test1(t *testing.T) {
// 	fmt.Println(errors.New("awd").Error() == errors.New("awd").Error())
// }

// func TestWS(t *testing.T) {
// 	ch := make(chan interface{})
// 	upgrader := websocket.Upgrader{}
// 	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		conn, err := upgrader.Upgrade(w, r, http.Header{})
// 		if err != nil {
// 			return
// 		}
// 		// conn.SetCloseHandler(func(code int, text string) error {
// 		// 	return fmt.Errorf("sss")
// 		// })
// 		for {
// 			_, p, err := conn.ReadMessage()
// 			if err != nil {
// 				fmt.Println(err)
// 				ch <- struct{}{}
// 				return
// 			}
// 			fmt.Println(p)
// 		}
// 	})
// 	go http.ListenAndServe("localhost:8080", nil)
// 	go func() {
// 		dialer := websocket.Dialer{}
// 		conn, _, err := dialer.Dial("ws://localhost:8080", http.Header{})
// 		// conn.SetCloseHandler(func(code int, text string) error {
// 		// 	return fmt.Errorf("zzz")
// 		// })
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 		go func() {
// 			for {
// 				_, _, err := conn.ReadMessage()
// 				if err != nil {
// 					fmt.Println(err)
// 					ch <- struct{}{}
// 					return
// 				}
// 				fmt.Println("!!")
// 			}
// 		}()
// 		for i := 0; i < 10; i++ {
// 			conn.WriteMessage(websocket.BinaryMessage, []byte{0, 1, 2})
// 		}
// 		now := time.Now()
// 		conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "awd"), now.Add(time.Second*5))
// 	}()
// 	<-ch
// 	<-ch
// }
