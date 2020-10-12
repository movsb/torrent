package message

import (
	"bytes"
	"crypto/sha1"
	"fmt"

	"github.com/movsb/torrent/tracker"
)

// Handshake ...
type Handshake struct {
	InfoHash  [sha1.Size]byte
	MyPeerID  tracker.PeerID
	HerPeerID tracker.PeerID
	data      []byte
}

var _ Marshaler = &Handshake{}
var _ Unmarshaler = &Handshake{}

var (
	handshakeStart    = byte(19)
	handshakeString   = `BitTorrent protocol`
	handshakeReserved = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}

	// HandshakeLength ...
	HandshakeLength        = 1 + len(handshakeString) + len(handshakeReserved) + sha1.Size + tracker.PeerIDLength
	handshakeInfoHashStart = HandshakeLength - sha1.Size - tracker.PeerIDLength
	handshakePeerIDStart   = HandshakeLength - tracker.PeerIDLength
)

func (m *Handshake) marshal() error {
	b := bytes.NewBuffer(nil)
	b.Grow(HandshakeLength)
	b.WriteByte(handshakeStart)
	b.WriteString(handshakeString)
	b.Write(handshakeReserved[:])
	b.Write(m.InfoHash[:])
	b.Write(m.MyPeerID[:])
	m.data = b.Bytes()
	return nil
}

// Marshal ...
func (m *Handshake) Marshal() ([]byte, error) {
	if len(m.data) == 0 {
		if err := m.marshal(); err != nil {
			return nil, err
		}
	}
	return m.data, nil
}

// Unmarshal ...
func (m *Handshake) Unmarshal(r []byte) error {
	if len(r) != HandshakeLength {
		return fmt.Errorf("handshake: length error")
	}

	start := handshakeInfoHashStart
	sent := m.data[start : start+sha1.Size]
	recv := r[start : start+sha1.Size]
	if !bytes.Equal(sent, recv) {
		return fmt.Errorf("handshake: info hash mismatch")
	}

	start = handshakePeerIDStart
	recv = r[start : start+tracker.PeerIDLength]
	if !bytes.Equal(m.HerPeerID[:], recv) {
		fmt.Println(m.HerPeerID)
		fmt.Println(recv)
		return fmt.Errorf("handshake: peer id mismatch")
	}

	return nil
}
