package message

import (
	"encoding/binary"
	"fmt"
)

// Have ...
type Have struct {
	Index int
}

var _ Message = &Have{}

// Marshal ...
func (m *Have) Marshal() ([]byte, error) {
	buf := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(buf, uint32(m.Index))
	return buf, nil
}

// Unmarshal ...
func (m *Have) Unmarshal(r []byte) error {
	if len(r) != 4 {
		return fmt.Errorf("message size should be 4")
	}
	m.Index = int(binary.BigEndian.Uint32(r))
	return nil
}
