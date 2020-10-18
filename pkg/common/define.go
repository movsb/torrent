package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zeebo/bencode"
)

// InfoHash ...
type InfoHash [20]byte

// Equal ...
func (ih InfoHash) Equal(other InfoHash) bool {
	return bytes.Equal(ih[:], other[:])
}

// Set ...
func (ih *InfoHash) Set(other []byte) {
	if len(other) != 20 {
		panic("info_hash length must be 20")
	}
	copy(ih[:], other)
}

// Copy ...
func (ih *InfoHash) Copy(other InfoHash) {
	copy(ih[:], other[:])
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
