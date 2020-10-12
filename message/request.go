package message

import (
	"encoding/binary"
	"fmt"
)

// MaxRequestLength ...
// All current implementations use 2^14 (16 kiB), and close connections
// which request an amount greater than that.
const MaxRequestLength = 16 << 10

// Request ...
type Request struct {
	Index  int
	Begin  int
	Length int
}

var _ Marshaler = &Request{}
var _ Unmarshaler = &Request{}

// Marshal ...
func (m *Request) Marshal() ([]byte, error) {
	buf := make([]byte, 4+4+4)
	binary.BigEndian.PutUint32(buf[0:], uint32(m.Index))
	binary.BigEndian.PutUint32(buf[4:], uint32(m.Begin))
	binary.BigEndian.PutUint32(buf[8:], uint32(m.Length))
	return buf, nil
}

// Unmarshal ...
func (m *Request) Unmarshal(r []byte) error {
	if len(r) != 4+4+4 {
		return fmt.Errorf("message size should be 12")
	}
	m.Index = int(binary.BigEndian.Uint32(r[0:]))
	m.Begin = int(binary.BigEndian.Uint32(r[4:]))
	m.Length = int(binary.BigEndian.Uint32(r[8:]))
	return nil
}
