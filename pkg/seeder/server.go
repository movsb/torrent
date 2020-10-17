package seeder

import (
	"fmt"
	"net"

	"github.com/movsb/torrent/file"
	"github.com/movsb/torrent/message"
	"github.com/movsb/torrent/peer"
)

// Server ...
type Server struct {
	Address string
}

// Run ...
func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	defer lis.Close()

	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	tf, err := file.ParseFile(`8ce301d28fe97eed1a6ef7feaf296411b375222f.torrent`)
	if err != nil {
		panic(err)
	}

	ifm := peer.NewIndexFileManager(tf.Name, tf.Single, tf.Files, tf.PieceLength, tf.PieceHashes)
	bf := message.NewBitField(tf.PieceHashes.Len(), 0xFF)

	c := peer.Client{
		Ifm:         ifm,
		MyBitField:  bf,
		HerBitField: message.NewBitField(tf.PieceHashes.Len(), 0),
		InfoHash:    tf.InfoHash(),
	}
	c.SetConn(conn)
	if err := c.Handshake2(); err != nil {
		panic(err)
	}

	if err := c.Send(message.MsgBitField, bf); err != nil {
		fmt.Printf("error send bf: %v\n", err)
		return
	}

	if err := c.Send(message.MsgUnChoke, message.UnChoke{}); err != nil {
		fmt.Printf("error send unchoked: %v\n", err)
	}
	if err := c.Send(message.MsgInterested, message.Interested{}); err != nil {
		fmt.Printf("error send unchoked: %v\n", err)
	}

	if err := c.Download(nil, nil); err != nil {
		fmt.Printf("error download: %v", err)
		return
	}
}
