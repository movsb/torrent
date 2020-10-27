package dht

import (
	"fmt"
	"testing"

	"github.com/movsb/torrent/pkg/common"
)

func TestPing(t *testing.T) {
	// node, _ := NodeIDFromString("05e2f6a48f1aaa04fe23a6e22f5c2c655590fd7a")
	dht := Client{
		MyNodeID: myNodeID,
		//Address:  `router.bittorrent.com:6881`,
		Address: `147.173.102.51:31851`,
	}
	ih, _ := common.HashFromString(`d1101a2b9d202811a05e8c57c557a20bf974dc8a`)
	//node, _ := NodeIDFromString(`d1101a2b9d202811a05e8c57c557a20bf974dc8a`)
	//node, _ := NodeIDFromString(`819e2e807aa72266abd8a8e1d11479feee667220`)
	_ = ih
	//fmt.Println(dht.GetPeers(ih))
	fmt.Println(dht.AnnouncePeer(ih, 6181, "\x31\x88\xe3\x3c"))
}
