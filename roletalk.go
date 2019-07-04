package roletalk

//DefaultPeer is default Peer instance
var DefaultPeer *Peer

func init() {
	DefaultPeer = NewPeer(PeerOptions{Name: "", Friendly: false})
}
