package dht

import (
	"fmt"
	"testing"
)

func TestPing(t *testing.T) {
	dht := Client{
		MyNodeID: myNodeID,
		Address:  `router.bittorrent.com:6881`,
	}
	fmt.Println(dht.Ping())
}
