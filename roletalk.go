package roletalk

var defaultPeer *Peer

//Singleton returns singleton Peer instance
func Singleton() *Peer {
	if defaultPeer == nil {
		defaultPeer = NewPeer(PeerOptions{Name: "", Friendly: false})
	}
	return defaultPeer
}
