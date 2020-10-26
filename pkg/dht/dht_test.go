package dht

import (
	"fmt"
	"testing"

	"github.com/movsb/torrent/pkg/common"
)

func TestPing(t *testing.T) {
	dht := Client{
		MyNodeID: myNodeID,
		Address:  `router.bittorrent.com:6881`,
	}
	hash, _ := common.HashFromString("\xd1\x10\x1a\x2b\x9d\x20\x28\x11\xa0\x5e\x8c\x57\xc5\x57\xa2\x0b\xf9\x74\xdc\x8a")
	fmt.Println(dht.GetPeers(hash))
}
