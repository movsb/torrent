package dht

import (
	"fmt"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	dht := Client{
		MyNodeID: myNodeID,
		Address:  `0.0.0.0:6181`,
	}
	go func() {
		if err := dht.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Second)
	//ih, _ := common.HashFromString(`d1101a2b9d202811a05e8c57c557a20bf974dc8a`)
	if err := dht.Ping(`router.bittorrent.com:6881`); err != nil {
		panic(err)
	}
	nodes, err := dht.FindNode(`router.bittorrent.com:6881`, myNodeID)
	if err != nil {
		panic(err)
	}
	for _, node := range nodes {
		fmt.Println(node)
	}
}
