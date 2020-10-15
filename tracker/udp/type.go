package tracker

import (
	"encoding/binary"
	"fmt"
	"net"

	tcptracker "github.com/movsb/torrent/tracker/tcp"
)

// Action ...
type Action uint32

// Actions ...
const (
	ActionConnect  = Action(0)
	ActionAnnounce = Action(1)
	ActionScrape   = Action(2)
	ActionError    = Action(3)
)

// Event ...
type Event uint32

// Events ...
const (
	EventNone      = Event(0)
	EventCompleted = Event(1)
	EventStarted   = Event(2)
	EventStopped   = Event(3)
)

const (
	protocolID = 0x41727101980
)

// ConnectRequest ...
type ConnectRequest struct {
	ProtocolID    uint64
	Action        Action
	TransactionID uint32
}

// Marshal ...
func (c ConnectRequest) Marshal() ([]byte, error) {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[0:], c.ProtocolID)
	binary.BigEndian.PutUint32(b[8:], uint32(c.Action))
	binary.BigEndian.PutUint32(b[12:], c.TransactionID)
	return b, nil
}

// ConnectResponse ...
type ConnectResponse struct {
	Action        Action
	TransactionID uint32
	ConnectionID  uint64
}

// Unmarshal ...
func (c *ConnectResponse) Unmarshal(b []byte) error {
	if len(b) < 16 {
		return fmt.Errorf("ConnectResponse: at least 16 bytes are required")
	}
	c.Action = Action(binary.BigEndian.Uint32(b[0:]))
	c.TransactionID = binary.BigEndian.Uint32(b[4:])
	c.ConnectionID = binary.BigEndian.Uint64(b[8:])
	return nil
}

// AnnounceRequest ...
type AnnounceRequest struct {
	ConnectionID  uint64
	Action        Action
	TransactionID uint32
	InfoHash      [20]byte
	PeerID        tcptracker.PeerID
	Downloaded    uint64
	Left          uint64
	Uploaded      uint64
	Event         Event
	IP            net.IP // IPv4 only
	Key           uint32
	NumWant       int32
	Port          uint16
}

// Marshal ...
func (r AnnounceRequest) Marshal() ([]byte, error) {
	b := make([]byte, 98)
	binary.BigEndian.PutUint64(b[0:], r.ConnectionID)
	binary.BigEndian.PutUint32(b[8:], uint32(r.Action))
	binary.BigEndian.PutUint32(b[12:], r.TransactionID)
	copy(b[16:], r.InfoHash[:])
	copy(b[36:], r.PeerID[:])
	binary.BigEndian.PutUint64(b[56:], r.Downloaded)
	binary.BigEndian.PutUint64(b[64:], r.Left)
	binary.BigEndian.PutUint64(b[72:], r.Uploaded)
	binary.BigEndian.PutUint32(b[80:], uint32(r.Event))
	copy(b[84:], r.IP.To4()[0:4]) // to  assert To4() doesn't return nil
	binary.BigEndian.PutUint32(b[88:], r.Key)
	binary.BigEndian.PutUint32(b[92:], uint32(r.NumWant))
	binary.BigEndian.PutUint16(b[96:], r.Port)
	return b, nil
}

// AnnounceResponse ...
type AnnounceResponse struct {
	Action        Action
	TransactionID uint32
	Interval      uint32
	Leechers      uint32
	Seeders       uint32
	Peers         []string
}

// Unmarshal ...
func (r *AnnounceResponse) Unmarshal(b []byte) error {
	if len(b) < 20 {
		return fmt.Errorf("AnnounceResponse: at least 20 bytes are required")
	}

	r.Action = Action(binary.BigEndian.Uint32(b[0:]))
	r.TransactionID = binary.BigEndian.Uint32(b[4:])
	r.Interval = binary.BigEndian.Uint32(b[8:])
	r.Leechers = binary.BigEndian.Uint32(b[12:])
	r.Seeders = binary.BigEndian.Uint32(b[16:])

	if len(b[20:])%6 != 0 {
		return fmt.Errorf("AnnounceResponse: malformed ip & port")
	}

	b = b[20:]
	for i, n := 0, len(b); i < n; i += 6 {
		ip := net.IPv4(b[i+0], b[i+1], b[i+2], b[i+3])
		port := int(b[i+4])<<8 + int(b[i+5])
		address := (&net.TCPAddr{IP: ip, Port: port}).String()
		r.Peers = append(r.Peers, address)
	}

	return nil
}
