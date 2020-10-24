package dht

import "net"

// Node ...
type Node struct {
	IP   net.IP
	Port uint16
	ID   NodeID
}
