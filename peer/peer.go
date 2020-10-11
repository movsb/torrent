package peer

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/movsb/torrent/message"
	"github.com/movsb/torrent/tracker"
)

// Client ...
type Client struct {
	Peer     tracker.Peer
	InfoHash [20]byte
	conn     net.Conn
	rw       *bufio.ReadWriter
}

// Close ...
func (c *Client) Close() error {
	c.rw.Flush()
	c.conn.Close()
	return nil
}

// Handshake ...
func (c *Client) Handshake() error {
	peerAddr := fmt.Sprintf("%s:%d", c.Peer.IP, c.Peer.Port)
	conn, err := net.DialTimeout("tcp", peerAddr, time.Second*3)
	if err != nil {
		return fmt.Errorf("client: handshake failed: %v", err)
	}
	c.conn = conn
	c.rw = bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)

	handshake := message.Handshake{
		InfoHash: c.InfoHash,
	}

	if err := c.roundtrip(&handshake, &handshake); err != nil {
		return fmt.Errorf("client: handshake failed: %v", err)
	}

	return nil
}

func (c *Client) roundtrip(req message.Marshaler, resp message.Unmarshaler) error {
	b, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("client: roundtrip: %v", err)
	}
	if _, err := c.rw.Write(b); err != nil {
		return fmt.Errorf("client: roundtrip: %v", err)
	}
	if err := c.rw.Flush(); err != nil {
		return fmt.Errorf("client: roundtrip: %v", err)
	}
	return resp.Unmarshal(c.rw)
}
