package peer

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/daemon/store"
	"github.com/movsb/torrent/pkg/message"
	"github.com/movsb/torrent/pkg/utils"
)

// SinglePieceData ...
type SinglePieceData struct {
	Index  int
	Hash   common.Hash
	Length int
	Data   []byte
}

// Peer ...
type Peer struct {
	Ctx       context.Context
	InfoHash  common.Hash
	HerPeerID common.PeerID
	PeerAddr  string
	conn      net.Conn

	rw *bufio.ReadWriter

	MyBitField  *message.BitField
	HerBitField *message.BitField

	PM *store.PieceManager

	msgch    chan message.Message
	HaveCh   chan int
	curPiece *SinglePieceData

	unchoked   bool
	downloaded int
	requested  int
	backlog    int
}

// tmp
func (c *Peer) SetConn(conn net.Conn) {
	c.conn = conn
	c.rw = bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)

	// TODO(movsb): init
	c.msgch = make(chan message.Message, 1)
	c.HaveCh = make(chan int, 3)
}

// Close ...
func (c *Peer) Close() error {
	if c.conn != nil {
		c.rw.Flush()
		c.conn.Close()
		c.conn = nil
	}
	return nil
}

// Send ...
func (c *Peer) Send(msgID message.MsgID, req message.Marshaler) error {
	b, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("client: send: %v", err)
	}
	sizeBuf := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(sizeBuf, 1+uint32(len(b)))
	utils.SetDeadlineSeconds(c.conn, 10)
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
func (c *Peer) Recv() (message.MsgID, message.Message, error) {
	sizeBuf := []byte{0, 0, 0, 0}
	utils.SetDeadlineSeconds(c.conn, 10)
	if _, err := io.ReadFull(c.rw, sizeBuf); err != nil {
		return 0, nil, fmt.Errorf("client: recv size: %v", err)
	}

	msgSize := binary.BigEndian.Uint32(sizeBuf)
	// keep alive
	if msgSize == 0 {
		log.Printf("keep alive from: %v", c.HerPeerID)
		return 0, nil, nil
	}
	if msgSize > 1<<20 {
		panic(`peer send large message size`)
	}

	buf := make([]byte, msgSize)
	utils.SetDeadlineSeconds(c.conn, 10)
	if _, err := io.ReadFull(c.rw, buf); err != nil {
		return 0, nil, fmt.Errorf("client: recv msg: %v", err)
	}

	var (
		msgID = message.MsgID(buf[0])
		msg   message.Message
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
		msg = &message.Request{}
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
func (c *Peer) RecvBitField() error {
keepalive:
	id, msg, err := c.Recv()
	if err != nil {
		log.Printf("recv bitfield failed: %v", err)
		return err
	}
	if msg == nil {
		goto keepalive
	}
	if id != message.MsgBitField {
		log.Printf("recv non-bitfield message: %v", id)
		return fmt.Errorf("recv non-bitfield message")
	}
	c.HerBitField = msg.(*message.BitField)
	c.HerBitField.Init(c.PM.PieceCount())
	return nil
}

// Download ...
func (c *Peer) Download(pending chan SinglePieceData, done chan SinglePieceData) error {
	errCh := make(chan error)
	go func() {
		errCh <- c.poll()
	}()

	for {
		select {
		case err := <-errCh:
			return err
		default:
		}

		piece, ok := c.getPendingPiece(pending)
		if !ok {
			return nil
		}

		c.curPiece = &piece
		if !c.HerBitField.HasPiece(piece.Index) {
			log.Printf("client %s doesn't have piece %d\n", c.HerPeerID, piece.Index)
			pending <- piece

			if have := c.MyBitField.HasPiece(piece.Index); have {
				if err := c.Send(message.MsgHave, &message.Have{Index: piece.Index}); err != nil {
					log.Printf("error send have: %v\n", err)
					return fmt.Errorf("peer: send have failed: %v", err)
				}
			}

			select {
			case msg := <-c.msgch:
				if err := c.readMessage(msg); err != nil {
					return err
				}
			case have := <-c.HaveCh:
				if err := c.Send(message.MsgHave, &message.Have{Index: have}); err != nil {
					log.Printf("error send have: %v\n", err)
					return err
				}
			default:
				time.Sleep(time.Millisecond * 100)
			}
			continue
		}

		if err := c.downloadPiece(&piece); err != nil {
			log.Printf("download piece failed: %v\n", err)
			pending <- piece
			return fmt.Errorf("download piece failed: %v", err)
		}

		if err := c.checkIntegrity(&piece); err != nil {
			log.Printf("check integrity failed: %v\n", err)
			pending <- piece
			return fmt.Errorf("check integrity failed: %v", err)
		}

		done <- piece
	}
}

func (c *Peer) poll() error {
	for {
		id, msg, err := c.Recv()
		if err != nil {
			log.Printf("client: read message failed: %v\n", err)
			return err
		}
		if msg == nil {
			log.Printf("keep alive\n")
			continue
		}
		_ = id
		c.msgch <- msg
	}
}

func (c *Peer) getPendingPiece(pending chan SinglePieceData) (SinglePieceData, bool) {
	select {
	case <-c.Ctx.Done():
		log.Printf("peer: context done")
		return SinglePieceData{}, false
	case <-time.After(time.Second * 3):
		log.Printf("peer: no piece to download for seconds. exiting.")
		return SinglePieceData{}, false
	case piece := <-pending:
		return piece, true
	}
}

func (c *Peer) downloadPiece(piece *SinglePieceData) error {
	c.requested = 0
	c.downloaded = 0
	c.backlog = 0

	if len(piece.Data) != piece.Length {
		piece.Data = make([]byte, piece.Length)
	}

	for c.downloaded < piece.Length {
		if c.unchoked {
			for c.backlog < 5 && c.requested < piece.Length {
				blockSize := message.MaxRequestLength
				if c.requested+blockSize > piece.Length {
					blockSize = piece.Length - c.requested
				}

				if err := c.Send(message.MsgRequest, &message.Request{
					Index:  piece.Index,
					Begin:  c.requested,
					Length: blockSize,
				}); err != nil {
					return fmt.Errorf("send request failed")
				}

				c.backlog++
				c.requested += blockSize
				//log.Printf("backlog: %d, requested: %d\n", c.backlog, c.requested)
			}
		}

		select {
		case msg := <-c.msgch:
			if err := c.readMessage(msg); err != nil {
				return err
			}
		case have := <-c.HaveCh:
			if err := c.Send(message.MsgHave, &message.Have{Index: have}); err != nil {
				log.Printf("error send have: %v\n", err)
				return err
			}
		}
	}

	return nil
}

func (c *Peer) readMessage(msg message.Message) error {
	piece := c.curPiece
	switch typed := msg.(type) {
	default:
		return fmt.Errorf("peer sent unknown message: %v", reflect.TypeOf(typed).String())
	case *message.Choke:
		c.unchoked = false
		log.Printf("peer choked\n")
	case *message.UnChoke:
		c.unchoked = true
		log.Printf("peer not choked\n")
	case *message.Interested:
		log.Printf("peer interested\n")
	case *message.Have:
		c.HerBitField.SetPiece(typed.Index)
		// log.Printf("peer has piece %d\n", typed.Index)
	case *message.Request:
		request := msg.(*message.Request)
		if !c.MyBitField.HasPiece(request.Index) {
			log.Printf("peer requests piece I don't have: %d\n", request.Index)
			return fmt.Errorf("peer: unexpected piece")
		}
		piece, err := c.PM.ReadPiece(request.Index)
		if err != nil {
			log.Printf("read piece failed: %d, %v\n", request.Index, err)
			return fmt.Errorf("peer: read piece failed: %v", err)
		}
		if request.Begin+request.Length > len(piece) {
			log.Printf("peer requests piece out of bound: %d\n", request.Index)
			return fmt.Errorf("peer: request piece out of bound")
		}
		if err := c.Send(message.MsgPiece, &message.Piece{
			Index: request.Index,
			Begin: request.Begin,
			Data:  piece[request.Begin : request.Begin+request.Length],
		}); err != nil {
			log.Printf("error sent piece: %v\n", err)
			return fmt.Errorf("peer: error sending piece: %v", err)
		}
		log.Printf("upload piece to %s: %d\n", c.PeerAddr, request.Index)
	case *message.Piece:
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
		c.downloaded += len(pieceRecv.Data)
		c.backlog--
	}

	return nil
}

func (c *Peer) checkIntegrity(piece *SinglePieceData) error {
	got := sha1.Sum(piece.Data)
	if !bytes.Equal(piece.Hash[:], got[:]) {
		return fmt.Errorf("check integrity failed")
	}
	return nil
}
