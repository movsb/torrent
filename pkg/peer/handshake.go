package peer

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/message"
	"github.com/movsb/torrent/pkg/utils"
)

// HandshakeOutgoing ...
func HandshakeOutgoing(conn net.Conn, timeout int, infoHash common.Hash, myPeerID common.PeerID) (*message.Handshake, error) {
	defer conn.SetDeadline(time.Time{})

	if err := handshakeSend(conn, timeout, infoHash, myPeerID); err != nil {
		return nil, err
	}
	m, err := handshakeRecv(conn, timeout)
	if err != nil {
		return nil, err
	}
	if !m.InfoHash.Equal(infoHash) {
		return nil, fmt.Errorf("handshake: info_hash mismatch")
	}

	// TODO(movsb): check peer id
	return m, nil
}

// HandshakeIncoming ...
func HandshakeIncoming(conn net.Conn, timeout int, myPeerID common.PeerID, onRecv func(*message.Handshake) error) (*message.Handshake, error) {
	defer conn.SetDeadline(time.Time{})

	m, err := handshakeRecv(conn, timeout)
	if err != nil {
		return nil, err
	}
	if err := onRecv(m); err != nil {
		return nil, err
	}
	if err := handshakeSend(conn, timeout, m.InfoHash, myPeerID); err != nil {
		return nil, err
	}

	return m, nil
}

func handshakeSend(conn net.Conn, timeout int, infoHash common.Hash, myPeerID common.PeerID) error {
	m := message.Handshake{
		InfoHash: infoHash,
		PeerID:   myPeerID,
	}
	b, err := m.Marshal()
	if err != nil {
		return fmt.Errorf("handshake: marshal failed: %v", err)
	}
	utils.SetDeadlineSeconds(conn, timeout)
	if _, err := conn.Write(b); err != nil {
		return fmt.Errorf("handshake: write failed: %v", err)
	}
	return nil
}

func handshakeRecv(conn net.Conn, timeout int) (*message.Handshake, error) {
	m := message.Handshake{}
	b := make([]byte, message.HandshakeLength)
	utils.SetDeadlineSeconds(conn, timeout)
	if _, err := io.ReadFull(conn, b); err != nil {
		return nil, fmt.Errorf("handshake: read failed: %v", err)
	}
	if err := m.Unmarshal(b); err != nil {
		return nil, fmt.Errorf("handshake: unmarshal failed: %v", err)
	}
	return &m, nil
}
