package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zeebo/bencode"
)

// Hash ...
type Hash [20]byte

// HashFromString ...
func HashFromString(s string) (Hash, error) {
	if len(s) != 20 {
		return Hash{}, fmt.Errorf("invalid string for info hash")
	}
	var hash Hash
	copy(hash[:], s)
	return hash, nil
}

func (h Hash) String() string {
	return fmt.Sprintf("%x", [20]byte(h))
}

// Equal ...
func (h *Hash) Equal(other Hash) bool {
	return bytes.Equal(h[:], other[:])
}

// Set ...
func (h *Hash) Set(other []byte) {
	if len(other) != 20 {
		panic("hash length must be 20")
	}
	copy(h[:], other)
}

// Copy ...
func (h *Hash) Copy(other Hash) {
	copy(h[:], other[:])
}

// MarshalBencode ...
func (h Hash) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes(string(h[:]))
}

// PieceHashes ...
type PieceHashes []byte

// Count ...
func (p PieceHashes) Count() int {
	return len(p) / 20
}

// Index ...
func (p PieceHashes) Index(index int) (h Hash) {
	if index < 0 || index > p.Count() {
		panic(fmt.Errorf("invalid piece index: %d", index))
	}

	s := 20 * index
	copy(h[:], p[s:s+20])
	return
}

// MarshalYAML ...
func (p PieceHashes) MarshalYAML() (interface{}, error) {
	list := make([]string, p.Count())
	for i := 0; i < p.Count(); i++ {
		list[i] = fmt.Sprintf("%x", p.Index(i))
	}
	return list, nil
}

// PeerID ...
type PeerID [PeerIDLength]byte

// MarshalBencode ...
func (p PeerID) MarshalBencode() ([]byte, error) {
	b := make([]byte, 3+PeerIDLength)
	b[0] = '2'
	b[1] = '0'
	b[2] = ':'
	copy(b[3:], p[:])
	return b, nil
}

func (p PeerID) String() string {
	b, _ := json.Marshal(string(p[:]))
	s := string(b[1 : len(b)-1])
	s = strings.ReplaceAll(s, `\"`, `"`)
	return s
}

// MarshalYAML ...
func (p PeerID) MarshalYAML() (interface{}, error) {
	return string(p.String()), nil
}

// PeerIDLength ...
const PeerIDLength = 20

// UnmarshalBencode ...
func (p *PeerID) UnmarshalBencode(r []byte) error {
	if len(r) != 3+PeerIDLength {
		return fmt.Errorf("peer id length error: %d != %d", len(r), PeerIDLength)
	}
	var id string
	if err := bencode.DecodeBytes(r, &id); err != nil {
		return fmt.Errorf("peer id error: %v", err)
	}
	copy(p[:], []byte(id))
	return nil
}

// Equal ...
func (p PeerID) Equal(other PeerID) bool {
	return bytes.Equal(p[:], other[:])
}
