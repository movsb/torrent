package dht

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	dht := New(myNodeID)
	dht.bootstrap()
	time.Sleep(time.Second)
	nodes, err := dht.findNodes(context.Background(), myNodeID)
	if err != nil {
		panic(err)
	}
	for _, node := range nodes {
		fmt.Println(node)
	}
}
