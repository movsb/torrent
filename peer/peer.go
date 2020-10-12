package peer

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
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
	bitField message.BitField
	unchoked bool
}

// Close ...
func (c *Client) Close() error {
	if c.conn != nil {
		c.rw.Flush()
		c.conn.Close()
		c.conn = nil
	}
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

// Send ...
func (c *Client) Send(msgID message.MsgID, req message.Marshaler) error {
	b, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	sizeBuf := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(sizeBuf, 1+uint32(len(b)))
	if _, err := c.rw.Write(sizeBuf); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	if err := c.rw.WriteByte(byte(msgID)); err != nil {
		return fmt.Errorf("client: send msg id: %v", err)
	}
	if _, err := c.rw.Write(b); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	if err := c.rw.Flush(); err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	return nil
}

// Recv ...
func (c *Client) Recv() (message.MsgID, message.Unmarshaler, error) {
	sizeBuf := []byte{0, 0, 0, 0}
	if _, err := io.ReadFull(c.rw, sizeBuf); err != nil {
		return 0, nil, fmt.Errorf("client: recv size: %v", err)
	}

	msgSize := binary.BigEndian.Uint32(sizeBuf)
	// keep alive
	if msgSize == 0 {
		log.Printf("keep alive from: %v", c.Peer)
		return 0, nil, nil
	}
	if msgSize > 1<<20 {
		panic(`peer send large message size`)
	}

	buf := make([]byte, msgSize)
	if _, err := io.ReadFull(c.rw, buf); err != nil {
		return 0, nil, fmt.Errorf("client: recv msg: %v", err)
	}

	var (
		msgID = message.MsgID(buf[0])
		msg   message.Unmarshaler
	)

	switch msgID {
	default:
		return 0, nil, fmt.Errorf("client: recv msg: unknown")
	case message.MsgChoke:
		msg = &message.Choke{}
	case message.MsgUnChoke:
		msg = &message.UnChoke{}
	case message.MsgInterested:
		msg = &message.Interested{}
	case message.MsgNotInterested:
		msg = &message.NotInterested{}
	case message.MsgHave:
		msg = &message.Have{}
	case message.MsgBitField:
		msg = &message.BitField{}
	case message.MsgRequest:
	case message.MsgPiece:
		msg = &message.Piece{}
	case message.MsgCancel:
	}

	if err := msg.Unmarshal(buf[1:]); err != nil {
		return 0, nil, fmt.Errorf("client: recv: unmarshal: %v", err)
	}

	return msgID, msg, nil
}

// RecvBitField ...
func (c *Client) RecvBitField() error {
	id, msg, err := c.Recv()
	if err != nil {
		log.Printf("recv bitfield failed: %v", err)
	}
	if id != message.MsgBitField {
		log.Printf("recv non-bitfield message: %v", id)
	}
	c.bitField = *msg.(*message.BitField)
	return nil
}

// Download ...
func (c *Client) Download(pending chan SinglePieceData, done chan SinglePieceData) error {
	for piece := range pending {
		if !c.bitField.HasPiece(piece.Index) {
			fmt.Printf("client %s doesn't have piece %d\n", c.Peer.ID, piece.Index)
			pending <- piece
			time.Sleep(time.Millisecond * 500)
			continue
		}

		if err := c.downloadPiece(&piece); err != nil {
			fmt.Printf("download piece failed: %v\n", err)
			pending <- piece
			time.Sleep(time.Millisecond * 500)
			continue
		}

		if err := c.checkIntegrity(&piece); err != nil {
			fmt.Printf("check integrity failed: %v\n", err)
			pending <- piece
			time.Sleep(time.Millisecond * 500)
			continue
		}

		c.Send(message.MsgHave, &message.Have{Index: piece.Index})

		done <- piece
	}

	return nil
}

func (c *Client) downloadPiece(piece *SinglePieceData) error {
	piece.requested = 0
	piece.downloaded = 0

	if len(piece.Data) != piece.Length {
		piece.Data = make([]byte, piece.Length)
	}

	for piece.downloaded < piece.Length {
		if c.unchoked {
			if piece.requested < piece.Length {
				blockSize := message.MaxRequestLength
				if piece.requested+blockSize > piece.Length {
					blockSize = piece.Length - piece.requested
				}

				if err := c.Send(message.MsgRequest, &message.Request{
					Index:  piece.Index,
					Begin:  piece.requested,
					Length: blockSize,
				}); err != nil {
					return fmt.Errorf("send request failed")
				}

				piece.requested += blockSize
			}
		}

		if err := c.readMessage(piece); err != nil {
			return fmt.Errorf("read message failed: %v", err)
		}
	}

	return nil
}

func (c *Client) readMessage(piece *SinglePieceData) error {
	id, msg, err := c.Recv()
	if err != nil {
		return fmt.Errorf("client: read message failed: %v", err)
	}
	if msg == nil {
		fmt.Printf("keep alive\n")
		return nil
	}

	switch id {
	default:
		return fmt.Errorf("peer sent unknown message: %v", id)
	case message.MsgChoke:
		c.unchoked = false
		fmt.Printf("peer choked\n")
	case message.MsgUnChoke:
		c.unchoked = true
		fmt.Printf("peer not choked\n")
	case message.MsgHave:
		have := msg.(*message.Have)
		c.bitField.SetPiece(have.Index)
		fmt.Printf("peer has piece %d\n", have.Index)
	case message.MsgPiece:
		pieceRecv := msg.(*message.Piece)
		if pieceRecv.Index != piece.Index {
			return fmt.Errorf("peer sent unknown piece index %d", pieceRecv.Index)
		}
		if pieceRecv.Begin < 0 || pieceRecv.Begin+len(pieceRecv.Data) > len(piece.Data) {
			return fmt.Errorf(
				"peer sent data too long: begin: %d + data: %d > data: %d",
				pieceRecv.Begin, len(pieceRecv.Data), len(piece.Data),
			)
		}
		copy(piece.Data[pieceRecv.Begin:], pieceRecv.Data)
		piece.downloaded += len(pieceRecv.Data)
	}

	return nil
}

func (c *Client) checkIntegrity(piece *SinglePieceData) error {
	got := sha1.Sum(piece.Data)
	if !bytes.Equal(piece.Hash, got[:]) {
		return fmt.Errorf("check integrity failed")
	}
	return nil
}
