package dht

import (
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net"

	"github.com/zeebo/bencode"
)

// NodeID ...
type NodeID [20]byte

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
	rand.Read(myNodeID[:])
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

// Ping ...
func (c *Client) Ping() (Response, error) {
	if err := c.dial(); err != nil {
		return Response{}, err
	}
	q := Query{
		KRPCCommon: KRPCCommon{
			TransactionID: makeTransactionID(),
			Type:          'q',
		},
		Query: `ping`,
		Args: map[string]interface{}{
			`id`: c.MyNodeID,
		},
	}
	b, err := bencode.EncodeBytes(q)
	if err != nil {
		return Response{}, err
	}
	if _, err := c.conn.Write(b); err != nil {
		log.Printf("dht: ping: %v", err)
		return Response{}, err
	}
	r := Response{}
	if err := bencode.NewDecoder(c.conn).Decode(&r); err != nil {
		return r, err
	}
	return r, nil
}
