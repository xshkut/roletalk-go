package roletalk

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type acquaintMsg struct {
	Address string   `json:"address"`
	ID      string   `json:"id"`
	Roles   []string `json:"roles"`
}

type rolesMsg struct {
	I     int      `json:"i"`
	Roles []string `json:"roles"`
}

func closeConnWithCode(conn *connLocker, code int, message string) error {
	return conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, message), time.Now().Add(heartBeatTimeout))
}

func (unit *Unit) sendRoles(i int, roles []string) error {
	str, err := json.Marshal(rolesMsg{i, roles})
	if err != nil {
		return err
	}
	_, err = unit.writeMsgToSomeConnection(serializeString(typeRoles, string(str)))
	return err
}

func (peer *Peer) introduceUnitToOthers(unit *Unit, addr string) error {
	if unit.friendly == false {
		return nil
	}
	units := peer.Units()
	am := unit.generateAcquaintMsg(addr)
	bin, err := json.Marshal(am)
	if err != nil {
		return err
	}
	for _, u := range units {
		if u.friendly == false || u == unit {
			continue
		}
		u.writeMsgToSomeConnection(append([]byte{typeAcquaint}, bin...))
	}
	return nil
}

func (unit *Unit) acquaintWithOthers() error {
	var err error
	if unit.friendly != true {
		return nil
	}
	for addr, u := range unit.peer.addrUnits.getUnitMap() {
		if unit == u || u == nil {
			continue
		}
		var b []byte
		am := u.generateAcquaintMsg(addr)
		b, err = json.Marshal(am)
		if err != nil {
			continue
		}
		_, err = unit.writeMsgToSomeConnection(append([]byte{typeAcquaint}, b...))
	}
	return err
}
