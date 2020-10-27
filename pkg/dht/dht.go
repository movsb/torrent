package dht

import (
	"encoding/hex"
	"fmt"
	"math/big"

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
