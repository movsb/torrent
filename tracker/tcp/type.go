package tracker

import (
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

func (p PeerID) String() string {
	b, _ := json.Marshal(p)
	s := string(b[1 : len(b)-1])
	s = strings.ReplaceAll(s, `\"`, `"`)
	return s
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
