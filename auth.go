package roletalk

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/pkg/errors"

	"github.com/gorilla/websocket"
)

type challengeWithIds struct {
	Challenge string   `json:"challenge"`
	Ids       []string `json:"ids"`
}

type proofWithID struct {
	Proof string `json:"proof"`
	ID    string `json:"id"`
}

type peerData struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	Friendly bool     `json:"friendly"`
	Meta     MetaInfo `json:"meta"`
}

//MetaInfo represents meta info of remote peer
type MetaInfo struct {
	Os       string `json:"os"`
	Runtime  string `json:"runtime"`
	Uptime   int64  `json:"uptime"`
	Time     int64  `json:"time"`
	Protocol string `json:"protocol"`
}

func (peer *Peer) authenticateWS(conn *connLocker) (peerData, error) {
	var success = make(chan peerData)
	var fail = make(chan error)
	data := peerData{}
	go func() {
		data, err := peer.startAuthWSHandshake(conn)
		if err != nil {
			fail <- err
		}
		success <- data
	}()
	select {
	case data := <-success:
		return data, nil
	case <-time.After(authTimeot):
		return data, fmt.Errorf("Auth timeout exceed: %v", authTimeot)
	case err := <-fail:
		return data, err
	}
}

func (peer *Peer) startAuthWSHandshake(conn *connLocker) (peerData, error) {
	confirmedIn := false
	confirmedOut := false
	var challenge string
	var err error
	res := peerData{}
	if len(peer.presharedKeys) > 0 {
		var msg string
		msg, challenge, err = peer.generateChallengeWithIds()
		if err != nil {
			return res, errors.Wrap(err, "Error while generating challenge for send")
		}
		chalPayload := append([]byte{byteAuthChallenge}, []byte(msg)...)
		if err := conn.WriteMessage(websocket.BinaryMessage, chalPayload); err != nil {
			return res, errors.Wrap(err, "Cannot send challenge to remote peer")
		}
	} else {
		confirmedIn = true
		pd, err := peer.generatePeerData()
		if err != nil {
			return res, errors.Wrap(err, "Cannot generate peer data to confirm auth for remote peer")
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, append([]byte{byteAuthConfirmed}, pd...)); err != nil {
			return res, errors.Wrap(err, "Cannot send confirmation to remote peer")
		}
	}
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			return res, errors.Wrap(err, "connection closed")
		}
		raw := p[1:]
		switch p[0] {
		case byteError:
			return res, fmt.Errorf("remote peer sent error: " + string(raw))
		case byteAuthConfirmed:
			confirmedOut = true
			if err := json.Unmarshal(raw, &res); err != nil {
				return res, errors.Wrap(err, "Error when parsing peer data: ")
			}
			if confirmedIn == true && confirmedOut == true {
				return res, nil
			}
		case byteAuthChallenge:
			proof, err := peer.processChallengeMsg(raw)
			if err != nil {
				return res, errors.Wrap(err, "Cannot process challenge message")
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, proof); err != nil {
				return res, errors.Wrap(err, "Cannot send generated proof")
			}
		case byteAuthResponse:
			if err := peer.verifyResponse(challenge, raw); err != nil {
				return res, errors.Wrap(err, "Response verification failed")
			}
			confirmedIn = true
			pd, err := peer.generatePeerData()
			if err != nil {
				return res, errors.Wrap(err, "Cannot generate peer data to confirm auth for remote peer")
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, append([]byte{byteAuthConfirmed}, pd...)); err != nil {
				return res, err
			}
			if confirmedIn == true && confirmedOut == true {
				return res, nil
			}
		}

	}
}

//AddKey ...
func (peer *Peer) AddKey(id, key string) {
	peer.presharedKeys = append(peer.presharedKeys, PresharedKey{id, key})
}

func (peer *Peer) generateChallengeWithIds() (msg, challenge string, err error) {
	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	challenge = fmt.Sprintf("%x", b)
	ids := make([]string, len(peer.presharedKeys))
	for i, pk := range peer.presharedKeys {
		ids[i] = pk.id
	}
	chalWithIds := challengeWithIds{challenge, ids}
	raw, err := json.Marshal(chalWithIds)
	if err != nil {
		return "", "", err
	}
	msg = string(raw)
	return
}

func (peer *Peer) processChallengeMsg(raw []byte) (res []byte, err error) {
	chalWithIds := challengeWithIds{}
	if err = json.Unmarshal(raw, &chalWithIds); err != nil {
		return nil, err
	}
	for _, preshared := range peer.presharedKeys {
		for _, id := range chalWithIds.Ids {
			if id == preshared.id {
				proof := peer.getProof(chalWithIds.Challenge, preshared.key)
				str, err := json.Marshal(proofWithID{proof, id})
				if err != nil {
					return nil, err
				}
				res = append([]byte{byteAuthResponse}, []byte(str)...)
				return res, nil
			}
		}
	}
	err = fmt.Errorf("Auth id mismatch")
	return
}

func (peer *Peer) getProof(challenge, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(challenge))
	return hex.EncodeToString(h.Sum(nil))
}

func (peer *Peer) verifyResponse(challenge string, raw []byte) error {
	proofAndID := proofWithID{}
	json.Unmarshal(raw, &proofAndID)
	for _, preshared := range peer.presharedKeys {
		if preshared.id == proofAndID.ID {
			originalHmac := hmac.New(sha256.New, []byte(preshared.key))
			originalHmac.Write([]byte(challenge))
			computed := hex.EncodeToString(originalHmac.Sum(nil))
			if computed != proofAndID.Proof {
				return fmt.Errorf("Hashes for proof with id %v are not equal", proofAndID.Proof)
			}
			return nil
		}
	}
	return fmt.Errorf("Response with id %v not found", proofAndID.ID)
}

func (peer *Peer) generatePeerData() ([]byte, error) {
	nowMs := int64(time.Now().UnixNano() / 10e6)
	meta := MetaInfo{Os: runtime.GOOS, Runtime: "GO", Time: nowMs, Uptime: int64(nowMs - peer.startTime.UnixNano()/10e6), Protocol: protocolVersion}
	pd := peerData{ID: peer.id, Friendly: peer.Friendly, Roles: peer.ListRoles(), Name: peer.Name, Meta: meta}
	marshaled, err := json.Marshal(pd)
	if err != nil {
		return nil, err
	}
	return marshaled, nil
}
