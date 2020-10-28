package dht

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net"

	"github.com/zeebo/bencode"
)

// NodeID ...
type NodeID [20]byte

// NodeIDFromString ...
func NodeIDFromString(s string) (NodeID, error) {
	if len(s) != 40 {
		return NodeID{}, fmt.Errorf(`node id string length is not equal to 40`)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return NodeID{}, err
	}
	var id NodeID
	copy(id[:], b)
	return id, nil
}

func (n NodeID) String() string {
	return fmt.Sprintf("%x", [20]byte(n))
}

// MarshalBencode ...
func (n NodeID) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes(string(n[:]))
}

// Distance ...
func (n NodeID) Distance(other NodeID) *big.Int {
	var xor NodeID
	for i := 0; i < len(n); i++ {
		xor[i] = n[i] ^ other[i]
	}
	return (&big.Int{}).SetBytes(xor[:])
}

var myNodeID NodeID

func init() {
	myNodeID, _ = NodeIDFromString(`d1101a2b9d202811a05e8c57c557a20bf974dc8a`)
}

// Node ...
type Node struct {
	IP   net.IP
	Port uint16
	ID   NodeID
}

// CompactNodeInfo ...
type CompactNodeInfo struct {
	ID   NodeID
	IP   net.IP
	Port uint16
}

// Unmarshal ...
func (n *CompactNodeInfo) Unmarshal(b []byte) error {
	if len(b) != 26 {
		return fmt.Errorf("_CompactNodeInfo: len != 26")
	}
	copy(n.ID[:], b[0:20])
	b = b[20:]
	n.IP = net.IPv4(b[0], b[1], b[2], b[3])
	b = b[4:]
	n.Port = uint16(b[0])<<8 + uint16(b[1])
	return nil
}

// CompactPeerInfo ...
type CompactPeerInfo struct {
	IP   net.IP
	Port uint16
}

// Unmarshal ...
func (n *CompactPeerInfo) Unmarshal(b []byte) error {
	if len(b) != 6 {
		return fmt.Errorf("_CompactPeerInfo: len != 6")
	}
	n.IP = net.IPv4(b[0], b[1], b[2], b[3])
	b = b[4:]
	n.Port = uint16(b[0])<<8 + uint16(b[1])
	return nil
}
