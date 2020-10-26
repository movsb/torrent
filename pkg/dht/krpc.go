package dht

import (
	"errors"
	"fmt"
	"log"
	"math/rand"

	"github.com/movsb/torrent/pkg/common"
	"github.com/zeebo/bencode"
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
	Code    _ErrorCode `bencode:"code"`
	Message string     `bencode:"message"`
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
	e.Code = _ErrorCode(code)
	message, ok := m[1].(string)
	if !ok {
		return fmt.Errorf("message is not a string")
	}
	e.Message = message
	return nil
}

type _ErrorCode int

// DHT Query Error
const (
	GenericError  _ErrorCode = 201
	ServerError   _ErrorCode = 202
	ProtocolError _ErrorCode = 203
	MethodUnknown _ErrorCode = 204
)

func (c *Client) send(query string, args map[string]interface{}) error {
	if err := c.dial(); err != nil {
		return err
	}

	q := Query{
		KRPCCommon: KRPCCommon{
			TransactionID: makeTransactionID(),
			Type:          'q',
		},
		Query: query,
		Args: map[string]interface{}{
			`id`: c.MyNodeID,
		},
	}
	for k, v := range args {
		q.Args[k] = v // will override existing args
	}

	b, err := bencode.EncodeBytes(q)
	if err != nil {
		return err
	}
	if _, err := c.conn.Write(b); err != nil {
		log.Printf("dht: find_node: %v", err)
		return err
	}

	return nil
}

func (c *Client) recv() (Response, error) {
	r := Response{}
	if err := bencode.NewDecoder(c.conn).Decode(&r); err != nil {
		return r, err
	}
	if r.Type == 'e' {
		return r, r.Err
	}
	if r.Type != 'r' {
		return r, fmt.Errorf("dht: response not r")
	}
	id, ok := r.Values[`id`].(string)
	if !ok {
		return r, fmt.Errorf(`dht: response has no id`)
	}
	if len(id) != 20 {
		return r, fmt.Errorf("dht: response id is not 20 bytes long")
	}

	return r, nil
}

// FindNode ...
func (c *Client) FindNode(target NodeID) ([]CompactNodeInfo, error) {
	args := map[string]interface{}{
		`target`: string(target[:]),
	}
	if err := c.send(`find_node`, args); err != nil {
		return nil, err
	}
	r, err := c.recv()
	if err != nil {
		return nil, fmt.Errorf("dht: recv: %v", err)
	}
	nodeValue, ok := r.Values[`nodes`]
	if !ok {
		return nil, nil // empty result isn't an error, the spec says 'should'
	}
	nodesString, ok := nodeValue.(string)
	if !ok {
		return nil, fmt.Errorf("dht: response nodes is not a string")
	}
	if len(nodesString)%26 != 0 {
		return nil, fmt.Errorf("dht: response nodes is not a multiply of 26 bytes")
	}

	nodeBytes := []byte(nodesString)
	nodes := make([]CompactNodeInfo, 0, len(nodeBytes)/26)

	for i := 0; i < len(nodeBytes); i += 26 {
		var node CompactNodeInfo
		if err := node.Unmarshal(nodeBytes[i : i+26]); err != nil {
			return nil, fmt.Errorf("dht: find_node: %v", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetPeers ...
func (c *Client) GetPeers(infoHash common.Hash) (token string, peers []CompactPeerInfo, nodes []CompactNodeInfo, rErr error) {
	args := map[string]interface{}{
		`info_hash`: infoHash,
	}
	if err := c.send(`get_peers`, args); err != nil {
		rErr = err
		return
	}
	r, err := c.recv()
	if err != nil {
		rErr = err
		return
	}

	tokenString, ok := r.Values[`token`].(string)
	if !ok {
		rErr = fmt.Errorf("dht: get_peers: no valid token returned")
		return
	}
	token = tokenString

	peerStrings, hasValues := r.Values[`values`].([]string)
	if hasValues {
		for _, value := range peerStrings {
			var peer CompactPeerInfo
			if err := peer.Unmarshal([]byte(value)); err != nil {
				rErr = fmt.Errorf("dht: get_peers: %v", err)
				return
			}
			peers = append(peers, peer)
		}
	}

	nodesString, hasNodes := r.Values[`nodes`].(string)
	if hasNodes {
		nodeBytes := []byte(nodesString)
		for i := 0; i < len(nodeBytes); i += 26 {
			var node CompactNodeInfo
			if err := node.Unmarshal(nodeBytes[i : i+26]); err != nil {
				rErr = fmt.Errorf("dht: find_node: %v", err)
				return
			}
			nodes = append(nodes, node)
		}
	}

	if (hasValues && hasNodes) || (!hasValues && !hasNodes) {
		rErr = fmt.Errorf("dht: get_peers: values and/or nodes are missing or exist both")
		return
	}

	return
}
