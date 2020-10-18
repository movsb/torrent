package seeder

import (
	"fmt"
	"net"

	"github.com/movsb/torrent/file"
	"github.com/movsb/torrent/peer"
	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/daemon/store"
	"github.com/movsb/torrent/pkg/message"
	trackercommon "github.com/movsb/torrent/pkg/tracker/common"
)

type LoadInfo struct {
	TF *file.File
	PM *store.PieceManager
	BF *message.BitField
}

type LoadTorrent interface {
	LoadTorrent(ih common.InfoHash) (*LoadInfo, error)
	AddClient(ih common.InfoHash, client *peer.Peer)
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
				fmt.Printf("self connect")
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
		fmt.Printf("HandshakeIncoming failed: %v", err)
		return
	}

	c := peer.Peer{
		HerPeerID:   handshake.PeerID,
		PM:          li.PM,
		MyBitField:  li.BF,
		HerBitField: message.NewBitField(li.TF.PieceHashes.Len(), 0),
		InfoHash:    li.TF.InfoHash(),
		PeerAddr:    conn.RemoteAddr().String(),
	}

	c.SetConn(conn)

	if err := c.Send(message.MsgBitField, li.BF); err != nil {
		fmt.Printf("error send bitbield: %v\n", err)
		return
	}
	if err := c.Send(message.MsgUnChoke, message.UnChoke{}); err != nil {
		fmt.Printf("error send unchoked: %v\n", err)
	}
	if err := c.Send(message.MsgInterested, message.Interested{}); err != nil {
		fmt.Printf("error send unchoked: %v\n", err)
	}

	closeConn = nil

	s.LoadTorrent.AddClient(handshake.InfoHash, &c)
}
