package message

import (
	"encoding/binary"
	"fmt"
)

// Piece ...
type Piece struct {
	Index int
	Begin int
	Data  []byte
}

var _ Message = &Request{}

// Marshal ...
func (m *Piece) Marshal() ([]byte, error) {
	buf := make([]byte, 4+4+len(m.Data))
	binary.BigEndian.PutUint32(buf[0:], uint32(m.Index))
	binary.BigEndian.PutUint32(buf[4:], uint32(m.Begin))
	copy(buf[8:], m.Data)
	return buf, nil
}

// Unmarshal ...
func (m *Piece) Unmarshal(r []byte) error {
	if len(r) < 8 {
		return fmt.Errorf("message size should be at least 8")
	}
	m.Index = int(binary.BigEndian.Uint32(r[0:]))
	m.Begin = int(binary.BigEndian.Uint32(r[4:]))
	m.Data = make([]byte, len(r)-8)
	copy(m.Data, r[8:])
	return nil
}
