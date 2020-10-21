package message

import (
	"log"
	"sync"
)

// BitField is just a byte array (bitmap) that stores bits which pieces a peer have.
// One bit per piece. If have, set to one. If not have, set to zero.
// BitField is shared between peers: downloaders & uploaders, so we need a lock.
// The bitsRemain is the bits that are not used for last byte.
type BitField struct {
	Fields []byte

	byteCount  int
	bitsRemain int

	mu sync.RWMutex
}

var _ Message = &BitField{}

// NewBitField ...
func NewBitField(pieceCount int, value byte) *BitField {
	bf := &BitField{}
	bf.Init(pieceCount)

	bytes := make([]byte, bf.byteCount)
	for i := 0; i < bf.byteCount; i++ {
		bytes[i] = value
	}

	bf.Fields = bytes

	return bf
}

// Marshal ...
// We make this a pointer receiver to avoid
// the copy of mutex.
func (m *BitField) Marshal() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.Fields, nil
}

// Unmarshal ...
func (m *BitField) Unmarshal(r []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Fields = make([]byte, len(r))
	copy(m.Fields, r)

	return nil
}

// Init ...
func (m *BitField) Init(pieceCount int) {
	byteCount := pieceCount / 8

	bitsRemain := 0
	if pieceCount%8 != 0 {
		byteCount++
		bitsRemain = 8 - pieceCount%8
	}

	m.byteCount = byteCount
	m.bitsRemain = bitsRemain
}

// HasPiece ...
func (m *BitField) HasPiece(index int) (has bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.calc(index, func(byteIndex int, bitMask byte) {
		has = m.Fields[byteIndex]&bitMask == bitMask
	})
	return
}

// SetPiece ...
func (m *BitField) SetPiece(index int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calc(index, func(byteIndex int, bitMask byte) {
		m.Fields[byteIndex] |= bitMask
	})
}

func (m *BitField) calc(index int, fn func(byteIndex int, bitMask byte)) {
	byteIndex := index / 8
	bitMask := byte(1 << (7 - index%8))

	if byteIndex < 0 || byteIndex >= m.byteCount {
		log.Printf(`invalid index: %d`, index)
		return
	}

	fn(byteIndex, bitMask)
}

// AllOnes indicates that we have all pieces.
func (m *BitField) AllOnes() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := 0; i < m.byteCount-1; i++ {
		if m.Fields[i] != 0xFF {
			return false
		}
	}

	lastByte := m.Fields[m.byteCount-1]
	return ^lastByte == 0xFF>>m.bitsRemain
}
