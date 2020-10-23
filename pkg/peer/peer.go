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

	OnExit func(p *Peer)

	conn net.Conn
	rw   *bufio.ReadWriter

	MyBitField  *message.BitField
	HerBitField *message.BitField

	PM *store.PieceManager

	msgCh  chan message.Message
	HaveCh chan int

	curPiece   *SinglePieceData
	chSetPiece chan *SinglePieceData
	donePiece  chan error

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

	c.chSetPiece = make(chan *SinglePieceData)
	c.msgCh = make(chan message.Message)
	c.HaveCh = make(chan int, 16)
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
	go c.Run()
	go c.poll()

	for {
		piece, ok := c.getPendingPiece(pending)
		if !ok {
			return nil
		}

		if !c.HerBitField.HasPiece(piece.Index) {
			log.Printf("client %s doesn't have piece %d\n", c.HerPeerID, piece.Index)
			pending <- piece

			// if have := c.MyBitField.HasPiece(piece.Index); have {
			// 	if err := c.Send(message.MsgHave, &message.Have{Index: piece.Index}); err != nil {
			// 		log.Printf("error send have: %v\n", err)
			// 		return fmt.Errorf("peer: send have failed: %v", err)
			// 	}
			// }

			time.Sleep(time.Second)

			continue
		}

		c.donePiece = make(chan error)
		c.chSetPiece <- &piece
		if err := <-c.donePiece; err != nil {
			log.Printf("download piece failed: %v\n", err)
			pending <- piece
			return fmt.Errorf("download piece failed: %v", err)
		}
		close(c.donePiece)

		if err := c.checkIntegrity(&piece); err != nil {
			log.Printf("check integrity failed: %v\n", err)
			pending <- piece
			return fmt.Errorf("check integrity failed: %v", err)
		}

		done <- piece
	}
}

func (c *Peer) getPendingPiece(pending chan SinglePieceData) (SinglePieceData, bool) {
	select {
	case <-c.Ctx.Done():
		log.Printf("peer: context done")
		return SinglePieceData{}, false
	case piece := <-pending:
		return piece, true
	}
}

func (c *Peer) exit(err error) {
	log.Printf("peer: exiting: %v", err)
	c.OnExit(c)
}

func (c *Peer) Run() {
	for {
		if c.curPiece != nil {
			if err := c.sendRequests(c.curPiece); err != nil {
				c.donePiece <- err
				return
			}
		}
		select {
		case msg := <-c.msgCh:
			if msg == nil {
				c.exit(fmt.Errorf("peer.work error on poll"))
				return
			}
			if err := c.handleMessage(msg); err != nil {
				c.exit(err)
				return
			}
		case have := <-c.HaveCh:
			if err := c.Send(message.MsgHave, &message.Have{Index: have}); err != nil {
				log.Printf("error send have: %v\n", err)
				c.exit(err)
				return
			}
		case piece := <-c.chSetPiece:
			c.reset(piece)
			c.curPiece = piece
		case <-c.Ctx.Done():
			log.Printf("peer.work: context done: %v", c.Ctx.Err())
			c.exit(c.Ctx.Err())
			return
		}
	}
}

func (c *Peer) reset(piece *SinglePieceData) {
	c.requested = 0
	c.downloaded = 0
	c.backlog = 0

	if len(piece.Data) != piece.Length {
		piece.Data = make([]byte, piece.Length)
	}
}

// poll polls messages from peer and sends it to
// the message channel. On error, the message will be nil.
func (c *Peer) poll() {
	for {
		_, msg, err := c.Recv()
		if err != nil {
			log.Printf("peer.poll failed: %v", err)
			c.msgCh <- nil
			return
		}
		if msg == nil {
			log.Printf("peer.poll keepalive from %s", c.PeerAddr)
			continue
		}
		select {
		case <-c.Ctx.Done():
			log.Printf("peer.poll context done: %v", c.Ctx.Err())
			return
		case c.msgCh <- msg:
		}
	}
}

func (c *Peer) sendRequests(piece *SinglePieceData) error {
	for c.unchoked && c.backlog < 5 && c.downloaded < piece.Length && c.requested < piece.Length {
		blockSize := message.MaxRequestLength
		if c.requested+blockSize > piece.Length {
			blockSize = piece.Length - c.requested
		}

		if err := c.Send(message.MsgRequest, &message.Request{
			Index:  piece.Index,
			Begin:  c.requested,
			Length: blockSize,
		}); err != nil {
			return fmt.Errorf("send request failed: %v", err)
		}

		c.backlog++
		c.requested += blockSize
	}
	return nil
}

func (c *Peer) handleMessage(msg message.Message) error {
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
		if piece == nil {
			return fmt.Errorf("peer sent piece but I'm downloading it")
		}
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
		if c.downloaded == piece.Length {
			c.donePiece <- nil
		}
		//log.Printf("receive piece: index=%d,begin:%d,length:%d backlog:%d,requested:%d,downloaded:%d",
		//	pieceRecv.Index, pieceRecv.Begin, len(pieceRecv.Data),
		//	c.backlog, c.requested, c.downloaded,
		//)
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
