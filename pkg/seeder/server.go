package seeder

import (
	"fmt"
	"log"
	"net"

	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/daemon/store"
	"github.com/movsb/torrent/pkg/message"
	"github.com/movsb/torrent/pkg/peer"
	"github.com/movsb/torrent/pkg/torrent"
	trackercommon "github.com/movsb/torrent/pkg/tracker/common"
)

type LoadInfo struct {
	TF *torrent.File
	PM *store.PieceManager
	BF *message.BitField
}

type LoadTorrent interface {
	LoadTorrent(ih common.Hash) (*LoadInfo, error)
	AddClient(ih common.Hash, client *peer.Peer)
}

// Server ...
type Server struct {
	Address  string
	MyPeerID common.PeerID

	LoadTorrent LoadTorrent
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
	closeConn := conn

	defer func() {
		if closeConn != nil {
			closeConn.Close()
		}
	}()

	var (
		err error
		li  *LoadInfo
	)

	handshake, err := peer.HandshakeIncoming(
		conn, 10, trackercommon.MyPeerID,
		func(m *message.Handshake) error {
			if s.MyPeerID.Equal(m.PeerID) {
				fmt.Printf("self connect\n")
				return fmt.Errorf("self connect")
			}
			li, err = s.LoadTorrent.LoadTorrent(m.InfoHash)
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Printf("HandshakeIncoming failed: %v\n", err)
		return
	}

	c := peer.Peer{
		HerPeerID:   handshake.PeerID,
		PM:          li.PM,
		MyBitField:  li.BF,
		HerBitField: message.NewBitField(li.TF.PieceHashes.Count(), 0),
		InfoHash:    li.TF.InfoHash(),
		PeerAddr:    conn.RemoteAddr().String(),
	}

	c.SetConn(conn)

	if err := c.Send(message.MsgBitField, li.BF); err != nil {
		log.Printf("error send bitbield: %v\n", err)
		return
	}
	if err := c.Send(message.MsgUnChoke, message.UnChoke{}); err != nil {
		log.Printf("error send unchoked: %v\n", err)
	}
	if err := c.Send(message.MsgInterested, message.Interested{}); err != nil {
		log.Printf("error send unchoked: %v\n", err)
	}

	closeConn = nil

	s.LoadTorrent.AddClient(handshake.InfoHash, &c)
}
