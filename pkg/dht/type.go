package dht

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
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

// Addr ...
func (n Node) Addr() string {
	return fmt.Sprintf("%s:%d", n.IP.String(), n.Port)
}

// CompactNodeInfo ...
type CompactNodeInfo Node

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

type _TransactionID [2]byte

// MarshalBencode ...
func (t _TransactionID) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes(string(t[:]))
}

// UnmarshalBencode ...
func (t *_TransactionID) UnmarshalBencode(b []byte) error {
	var s string
	if err := bencode.DecodeBytes(b, &s); err != nil {
		return err
	}
	if len(s) != 2 {
		return errors.New("transaction id len != 2")
	}
	t[0], t[1] = s[0], s[1]
	return nil
}

type _ByteAsString byte

// MarshalBencode ...
func (b _ByteAsString) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes(string(b))
}

// UnmarshalBencode ...
func (b *_ByteAsString) UnmarshalBencode(r []byte) error {
	var s string
	if err := bencode.DecodeBytes(r, &s); err != nil {
		return err
	}
	if len(s) != 1 {
		return errors.New("len != 1")
	}
	*b = _ByteAsString(s[0])
	return nil
}

// MakeTransactionID ...
func makeTransactionID() (t _TransactionID) {
	rand.Read(t[:])
	return
}

// Message ...
type Message struct {
	TransactionID _TransactionID         `bencode:"t"`
	Type          _ByteAsString          `bencode:"y"`
	Query         string                 `bencode:"q,omitempty"`
	Args          map[string]interface{} `bencode:"a,omitempty"`
	Values        map[string]interface{} `bencode:"r,omitempty"`
	Err           *_E                    `bencode:"e,omitempty"`
}

type _E struct {
	Code    int    `bencode:"code"`
	Message string `bencode:"message"`
}

func (e _E) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// UnmarshalBencode ...
func (e *_E) UnmarshalBencode(l []byte) error {
	var hp interface{}
	if err := bencode.DecodeBytes(l, &hp); err != nil {
		return err
	}
	m, ok := hp.([]interface{})
	if !ok {
		return fmt.Errorf("e isn't a list")
	}
	code, ok := m[0].(int64)
	if !ok {
		return fmt.Errorf("code is not an integer")
	}
	e.Code = int(code)
	message, ok := m[1].(string)
	if !ok {
		return fmt.Errorf("message is not a string")
	}
	e.Message = message
	return nil
}
