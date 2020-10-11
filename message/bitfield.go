package message

import (
	"fmt"
	"log"
)

// BitField ...
type BitField struct {
	Fields []byte
}

var _ Marshaler = &BitField{}
var _ Unmarshaler = &BitField{}

// Marshal ...
func (m *BitField) Marshal() ([]byte, error) {
	return nil, fmt.Errorf(`not implemented`)
}

// Unmarshal ...
func (m *BitField) Unmarshal(r []byte) error {
	m.Fields = make([]byte, len(r))
	copy(m.Fields, r)
	return nil
}

// HasPiece ...
func (m *BitField) HasPiece(index int) (has bool) {
	m.calc(index, func(byteIndex int, bitMask byte) {
		has = m.Fields[byteIndex]&bitMask == bitMask
	})
	return
}

// SetPiece ...
func (m *BitField) SetPiece(index int) {
	m.calc(index, func(byteIndex int, bitMask byte) {
		m.Fields[byteIndex] |= bitMask
	})
}

func (m *BitField) calc(index int, fn func(byteIndex int, bitMask byte)) {
	byteIndex := index / 8
	bitMask := byte(1 << (7 - index%8))

	if byteIndex < 0 || byteIndex >= len(m.Fields) {
		log.Printf(`invalid index: %d`, index)
		return
	}

	fn(byteIndex, bitMask)
}
