package utils

import (
	"net"
	"time"
)

// SetDeadlineSeconds ...
func SetDeadlineSeconds(conn net.Conn, seconds int) error {
	return conn.SetDeadline(time.Now().Add(time.Second * time.Duration(seconds)))
}
