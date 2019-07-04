package roletalk

// func TestWithEcho(t *testing.T) {
// 	address := "ws://localhost:5000"
// 	fmt.Printf("Testing with echo server. Assuming it is running on %v", address)
// 	peer := NewPeer(PeerOptions{})
// 	peer.preshareKey("echo", "echo key")
// 	unit, err := peer.Connect(address, ConnectOptions{allowUntrustedTLS: true, reconnect: true})
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
// 	fmt.Printf("\n%+v\n", unit.ID)
// 	if err := unit.send(headersForSendable{event: "echo", role: "echo"}, 1); err != nil {
// 		t.Error(err.Error())
// 	}
// }
