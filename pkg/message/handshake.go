package message

import (
	"bytes"
	"crypto/sha1"
	"fmt"

	"github.com/movsb/torrent/pkg/common"
)

// Handshake ...
type Handshake struct {
	InfoHash common.Hash
	PeerID   common.PeerID
}

// Handshake indeed is not a BitTorrent message.
// We just put it here for convenience.
var _ Message = &Handshake{}

var (
	handshakeStart    = byte(19)
	handshakeString   = `BitTorrent protocol`
	handshakeReserved = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}

	// HandshakeLength ...
	HandshakeLength        = 1 + len(handshakeString) + len(handshakeReserved) + sha1.Size + common.PeerIDLength
	handshakeInfoHashStart = HandshakeLength - sha1.Size - common.PeerIDLength
	handshakePeerIDStart   = HandshakeLength - common.PeerIDLength
)

// Marshal ...
func (m *Handshake) Marshal() ([]byte, error) {
	b := bytes.NewBuffer(nil)
	b.Grow(HandshakeLength)
	b.WriteByte(handshakeStart)
	b.WriteString(handshakeString)
	b.Write(handshakeReserved[:])
	b.Write(m.InfoHash[:])
	b.Write(m.PeerID[:])
	return b.Bytes(), nil
}

// Unmarshal ...
func (m *Handshake) Unmarshal(r []byte) error {
	if len(r) != HandshakeLength {
		return fmt.Errorf("handshake: invalid length")
	}

	if startChar := r[0]; startChar != handshakeStart {
		return fmt.Errorf("handshake: invalid start character: %d", startChar)
	}
	if btProto := string(r[1 : 1+19]); btProto != handshakeString {
		return fmt.Errorf("handshake: invalid protocol: %s", btProto)
	}
	if reserved := r[20 : 20+8]; !bytes.Equal(reserved, handshakeReserved[:]) {
		fmt.Printf("handshake: invalid reserved: %x\n", reserved)
		// return fmt.Errorf("handshake: invalid reserved: %x", reserved)
	}

	start := handshakeInfoHashStart
	copy(m.InfoHash[:], r[start:start+sha1.Size])

	start = handshakePeerIDStart
	copy(m.PeerID[:], r[start:start+common.PeerIDLength])

	return nil
}
