package message

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
)

// Handshake ...
type Handshake struct {
	InfoHash [sha1.Size]byte
	data     []byte
}

var _ Marshaler = &Handshake{}
var _ Unmarshaler = &Handshake{}

var (
	handshakeStart    = byte(19)
	handshakeString   = `BitTorrent protocol`
	handshakeReserved = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}

	handshakeLength        = 1 + len(handshakeString) + len(handshakeReserved) + sha1.Size
	handshakeInfoHashStart = handshakeLength - sha1.Size
)

func (m *Handshake) marshal() error {
	b := bytes.NewBuffer(nil)
	b.Grow(handshakeLength)
	b.WriteByte(handshakeStart)
	b.WriteString(handshakeString)
	b.Write(handshakeReserved[:])
	b.Write(m.InfoHash[:])
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
func (m *Handshake) Unmarshal(r io.Reader) error {
	b := make([]byte, handshakeLength)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return fmt.Errorf("handshake: read failed: %v", err)
	}

	start := handshakeInfoHashStart
	sent := m.data[start : start+sha1.Size]
	recv := b[start : start+sha1.Size]
	if !bytes.Equal(sent, recv) {
		return fmt.Errorf("handshake: mismatch")
	}

	return nil
}
