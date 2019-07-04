package main

import (
	"fmt"
	"log"
	"roletalk"
)

var peer *roletalk.Peer

func main() {
	peer = roletalk.DefaultPeer
	// peer.AddKey("echo", "echo key")
	if _, err := peer.Connect("ws://localhost:5000"); err != nil {
		log.Fatal(err)
	}
	ctx, err := peer.Destination("echo").Request("echo", roletalk.EmitOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ctx.Data)
}
