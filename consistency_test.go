package roletalk

import (
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/assert"
)

var peer1 = NewPeer(PeerOptions{Name: "test peer 1", Friendly: true})
var peer2 = NewPeer(PeerOptions{Name: "test peer 2", Friendly: true})
var peer3 = NewPeer(PeerOptions{Name: "test peer 3", Friendly: true})

var address1 string
var address3 string

var role1 = "initial_role_1"
var role2 = "initial_role_2"
var role3 = "initial_role_3"
var dynamicRole = "dynamicly_added_role"

var peerOnUnited uint32
var destOnUnited uint32
var destOnClosed uint32
var unitOnClosed uint32
var peerOnRoled uint32
var RoleOnStatusChanged int

func TestUnderlyingCommunication(t *testing.T) {
	for _, peer := range []*Peer{peer1, peer2, peer3} {
		peer.AddKey("awd", "zxc")
	}
	peer1.OnRole(func(role *Role) {
		if role.Name() == role1 {
			atomic.AddUint32(&peerOnRoled, 1)
		}
	})
	peer1.Role(role1).OnStatusChange(func() {
		RoleOnStatusChanged++
	})
	peer1.Role(role1)
	peer1.Role(role1)
	peer2.Role(role2)
	peer3.Role(role3)
	peer2.Destination(role1)
	peer2.Destination(dynamicRole)
	peer1.Destination(role3)
	if addr, err := peer1.Listen("localhost:0"); err != nil {
		t.Error("Cannot start tests. Cannot listen")
	} else {
		address1 = addr.String()
	}
	if addr, err := peer3.Listen("localhost:0"); err != nil {
		t.Error("Cannot start tests. Cannot listen")
	} else {
		address3 = addr.String()
	}
	peer2.Destination(role1).OnUnit(func(u *Unit) {
		atomic.StoreUint32(&destOnUnited, 1)
	})
	peer2.OnUnit(func(u *Unit) {
		atomic.StoreUint32(&peerOnUnited, 1)
	})
	peer1.Role("role1")
	t.Run("testing Peer.Connect", connectPeers)
	time.Sleep(time.Millisecond * 10)
	t.Run("testing initial destination consistency", testInitialRoles)
	t.Run("testing dynamic destination consistency", testDynamicRoles)
	t.Run("testing aqcuaint logic", testAcquaint)
	t.Run("testing reconnection after conn abort", testReconnectionAbort)
	t.Run("testing reconnections after manual close", testReconnectionManual)
	peer1.Close()
}

func connectPeers(t *testing.T) {
	if _, err := peer2.Connect("ws://"+address1, ConnectOptions{}); err != nil {
		t.Error(err)
	}
	if _, err := peer2.Connect("ws://"+address3, ConnectOptions{}); err != nil {
		t.Error(err)
	}
}

func testInitialRoles(t *testing.T) {
	assert.Equal(t, atomic.LoadUint32(&peerOnRoled), uint32(1))
	assert.Equal(t, atomic.LoadUint32(&destOnUnited), uint32(1))
	assert.Equal(t, atomic.LoadUint32(&peerOnUnited), uint32(1))

	peer2dest1Ready := peer2.Destination(role1).Ready()
	assert.Equal(t, peer2dest1Ready, true)
	assert.Equal(t, peer2.Destination(dynamicRole).Ready(), false)
	_, hasDest := peer2.destinations[role3]
	assert.Equal(t, hasDest, false)
	peer2.Destination(role3)
	_, hasDest = peer2.destinations[role3]
	assert.Equal(t, hasDest, true)
	assert.Equal(t, peer2.Destination(role3).Ready(), true)

	peer1.Role(role1).Enable()
	assert.Equal(t, RoleOnStatusChanged, 0)
	peer1.Role(role1).Disable()
	assert.Equal(t, RoleOnStatusChanged, 1)
	peer1.Role(role1).Disable()
	assert.Equal(t, RoleOnStatusChanged, 1)
	peer1.Role(role1).Enable()
	assert.Equal(t, RoleOnStatusChanged, 2)
}

func testDynamicRoles(t *testing.T) {
	time.Sleep(time.Second)
	assert.Equal(t, peer2.Destination(dynamicRole).Ready(), false)
	// fmt.Println("FROM HERE")
	peer1.Role(dynamicRole)
	// fmt.Printf("peer1: %v, peer2: %v\n", peer1.ID(), peer2.ID())
	time.Sleep(time.Millisecond * 5)
	// fmt.Println("TO HERE")
	assert.Equal(t, peer2.Destination(dynamicRole).Ready(), true)
	peer1.Role(dynamicRole).Disable()
	time.Sleep(time.Millisecond * 5)
	assert.Equal(t, peer2.Destination(dynamicRole).Ready(), false)
}

func testAcquaint(t *testing.T) {
	assert.Equal(t, peer1.Destination(role3).Ready(), true)
}

func testReconnectionAbort(t *testing.T) {
	//Should reconnect if connection was aborted on the socket listener's side

	for _, unit := range peer1.Destination(role2).Units() {
		unit.Close()
	}
	assert.Equal(t, peer1.Destination(role2).Ready(), false)

	time.Sleep(time.Millisecond * 40)
	assert.Equal(t, peer1.Destination(role2).Ready(), true)
}

func testReconnectionManual(t *testing.T) {
	//Should not reconnect if connection was manually closed locally
	assert.Equal(t, peer1.Destination(role2).Ready(), true)
	peer1.Destination(role2).OnClose(func() {
		atomic.StoreUint32(&destOnClosed, 1)
	})
	for _, unit := range peer2.Destination(role1).Units() {
		unit.OnClose(func(err error) {
			atomic.StoreUint32(&unitOnClosed, 1)
		})
		unit.Close()
	}
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, atomic.LoadUint32(&unitOnClosed), uint32(1))
	assert.Equal(t, atomic.LoadUint32(&destOnClosed), uint32(1))
	assert.Equal(t, peer1.Destination(role2).Ready(), false)
}
