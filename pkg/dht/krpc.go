package dht

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

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

// Server ...
type Server struct {
	MyNodeID NodeID
	Address  string
}

// Serve ...
func (s *Server) Serve() error {
	addr, err := net.ResolveUDPAddr(`udp`, s.Address)
	if err != nil {
		return fmt.Errorf(`dht: server: serve: ResolveUDPAddr: %v`, err)
	}

	conn, err := net.ListenUDP(`udp`, addr)
	if err != nil {
		return fmt.Errorf(`dht: server: serve: ListenUDP: %v`, err)
	}
	defer conn.Close()

	buf := make([]byte, 64<<10)
	for {
		n, r, err := conn.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf(`dht: server: serve: ReadFromUDP: %v`, err)
		}
		s.recv(r, buf[:n])
	}
}

func (s *Server) recv(addr *net.UDPAddr, buf []byte) {

}

// Ping ...
func (s *Server) Ping(q *Query) {

}

// FindNode ...
func (s *Server) FindNode(q *Query) {

}

// GetPeers ...
func (s *Server) GetPeers(q *Query) {

}

// AnnouncePeer ...
func (s *Server) AnnouncePeer(q *Query) {

}

// Client ...
type Client struct {
	MyNodeID NodeID
	Address  string

	conn *net.UDPConn
}

func (c *Client) dial() error {
	if c.conn != nil {
		return nil
	}
	dstAddr, err := net.ResolveUDPAddr("udp", c.Address)
	if err != nil {
		return fmt.Errorf("resolve udp address failed: %v", err)
	}
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		return fmt.Errorf("dial udp address failed: %v", err)
	}
	c.conn = conn
	return nil
}

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
	fmt.Println(`send:`, string(b))
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
	defer c.conn.SetReadDeadline(time.Time{})
	c.conn.SetReadDeadline(time.Now().Add(time.Second * 5))
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

// Ping ...
func (c *Client) Ping() error {
	if err := c.send(`ping`, nil); err != nil {
		return err
	}
	r, err := c.recv()
	if err != nil {
		return nil
	}
	_ = r
	return nil
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
	token = fmt.Sprintf("%x", tokenString)

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

// AnnouncePeer ...
func (c *Client) AnnouncePeer(infoHash common.Hash, port uint16, token string) error {
	args := map[string]interface{}{
		`info_hash`: infoHash,
		`port`:      port,
		`token`:     token,
	}
	if err := c.send(`announce_peer`, args); err != nil {
		return err
	}
	r, err := c.recv()
	if err != nil {
		return err
	}
	_ = r
	return nil
}
