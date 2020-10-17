package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zeebo/bencode"
)

// AnnounceResponse ...
type AnnounceResponse struct {
	FailureReason string `bencode:"failure reason"`
	Interval      int    `bencode:"interval"`
	Peers         []Peer `bencode:"peers"`
}

// Peer ...
type Peer struct {
	ID   PeerID `bencode:"peer id"`
	IP   string `bencode:"ip"`
	Port int    `bencode:"port"`
}

func (p Peer) String() string {
	return fmt.Sprintf(`%v (%s:%d)`, p.ID.String(), p.IP, p.Port)
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
