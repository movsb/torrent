package dht

import (
	"fmt"
	"net"
)

// CompactNodeInfo ...
type CompactNodeInfo struct {
	ID   NodeID
	IP   net.IP
	Port uint16
}

// Unmarshal ...
func (n *CompactNodeInfo) Unmarshal(b []byte) error {
	if len(b) != 26 {
		return fmt.Errorf("_CompactNodeInfo: len != 26")
	}
	copy(n.ID[:], b[0:20])
	b = b[20:]
	n.IP = net.IPv4(b[0], b[1], b[2], b[3])
	b = b[4:]
	n.Port = uint16(b[0])<<8 + uint16(b[1])
	return nil
}

// CompactPeerInfo ...
type CompactPeerInfo struct {
	IP   net.IP
	Port uint16
}

// Unmarshal ...
func (n *CompactPeerInfo) Unmarshal(b []byte) error {
	if len(b) != 6 {
		return fmt.Errorf("_CompactPeerInfo: len != 6")
	}
	n.IP = net.IPv4(b[0], b[1], b[2], b[3])
	b = b[4:]
	n.Port = uint16(b[0])<<8 + uint16(b[1])
	return nil
}
