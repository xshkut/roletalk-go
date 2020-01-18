package roletalk

var defaultPeer *Peer

//Singleton returns singleton Peer instance. It is used to share single Peer between multiple places of your code
func Singleton() *Peer {
	if defaultPeer == nil {
		defaultPeer = NewPeer(PeerOptions{Name: "", Friendly: false})
	}
	return defaultPeer
}
