package peer

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
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
		InfoHash:  c.InfoHash,
		MyPeerID:  tracker.MyPeerID,
		HerPeerID: c.Peer.ID,
	}

	b, err := handshake.Marshal()
	if err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	if _, err := c.rw.Write(b); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	if err := c.rw.Flush(); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	b = make([]byte, message.HandshakeLength)
	if _, err = io.ReadFull(c.rw, b); err != nil {
		return fmt.Errorf("client: read handshake: %v", err)
	}
	if err := handshake.Unmarshal(b); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	return nil
}

func (c *Client) Send(req message.Marshaler) error {
	b, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	sizeBuf := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(sizeBuf, uint32(len(b)))
	if _, err := c.rw.Write(sizeBuf); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	if _, err := c.rw.Write(b); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	if err := c.rw.Flush(); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	return nil
}

func (c *Client) Recv(resp message.Unmarshaler) error {
	sizeBuf := []byte{0, 0, 0, 0}
	if _, err := io.ReadFull(c.rw, sizeBuf); err != nil {
		return fmt.Errorf("client: recv: %v", err)
	}
	msgSize := binary.BigEndian.Uint32(sizeBuf)
	if msgSize > 1<<20 {
		panic(`peer send large message size`)
	}
	buf := make([]byte, msgSize)
	if _, err := io.ReadFull(c.rw, buf); err != nil {
		return fmt.Errorf("client: recv: %v", err)
	}
	if err := resp.Unmarshal(buf); err != nil {
		return fmt.Errorf("client: recv: unmarshal: %v", err)
	}
	return nil
}
