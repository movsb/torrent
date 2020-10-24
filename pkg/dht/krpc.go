package dht

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/zeebo/bencode"
)

// KRPCMethod ...
type KRPCMethod byte

// KRPCID ...
type KRPCID [20]byte

// KRPC messages ...
const (
	Ping KRPCMethod = iota
	FindNode
	GetPeers
	AnnouncePeers
)

// KRPCCommon ...
type KRPCCommon struct {
	TransactionID _TransactionID `bencode:"t"` // the transaction id
	Type          _ByteAsString  `bencode:"y"` // the type of message: `q` for query, `r` for response, `e` for error
}

type _TransactionID [2]byte

// MarshalBencode ...
func (t _TransactionID) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes(string(t[:]))
}

// UnmarshalBencode ...
func (t _TransactionID) UnmarshalBencode(b []byte) error {
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

// Query ...
type Query struct {
	KRPCCommon
	Query string                 `bencode:"q"`
	Args  map[string]interface{} `bencode:"a"`
}

// Response ...
type Response struct {
	KRPCCommon
	Values map[string]interface{} `bencode:"r"`
	Err    _E                     `bencode:"e"`
}

type _E struct {
	Code    int    `bencode:"code"`
	Message string `bencode:"message"`
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
