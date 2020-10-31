package dht

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// Some constants
const (
	K = 100 // The replication factor for bucket.
	A = 8   // α, alpha, not a. The concurency parameter.
)

// DHT ...
type DHT struct {
	router *Router
	client *Client
}

// New ...
func New(myID NodeID) *DHT {
	dht := &DHT{
		router: NewRouter(myID),
		client: &Client{
			MyNodeID: myID,
			Address:  `0.0.0.0:6181`,
		},
	}
	go func() {
		if err := dht.client.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Second)
	return dht
}

func (dht *DHT) bootstrap() {
	initial := []string{
		`router.bittorrent.com:6881`,
	}

	wg := &sync.WaitGroup{}
	for _, boot := range initial {
		wg.Add(1)
		go func(boot string) {
			defer wg.Done()
			node, err := dht.client.Ping(boot)
			if err != nil {
				log.Printf("dht: bootstrap: %v", err)
				return
			}
			dht.router.Upsert(node, false)
		}(boot)
	}
	wg.Wait()

	nodes, err := dht.findNodes(context.TODO(), dht.router.myID)
	if err != nil {
		log.Printf("dht: bootstrap: %v", err)
		return
	}

	for _, node := range nodes {
		dht.router.Upsert(node, false)
	}
}

func (dht *DHT) findNodes(ctx context.Context, target NodeID) ([]Node, error) {
	nodes := dht.router.Closest(target, A)
	if len(nodes) == 0 {
		return nil, fmt.Errorf("dht: no nodes")
	}

	mu := &sync.Mutex{}
	visited := make(map[NodeID]bool)
	newNodes := make([]Node, len(nodes))
	copy(newNodes, nodes)

	for len(newNodes) > 0 {
		wg := &sync.WaitGroup{}
		wp := make(chan struct{}) // write protect
		for _, node := range newNodes {
			wg.Add(1)
			go func(node Node) {
				defer wg.Done()
				foundNodes, err := dht.client.FindNode(node.Addr(), target)
				if err != nil {
					log.Printf("dht: get_peers: %v", err)
					return
				}
				<-wp
				mu.Lock()
				defer mu.Unlock()
				visited[node.ID] = true
				for _, node := range foundNodes {
					if !visited[node.ID] {
						newNodes = append(newNodes, Node(node))
						nodes = append(nodes, Node(node))
					}
				}
			}(node)
		}
		newNodes = newNodes[:0]
		close(wp)
		wg.Wait()
		log.Printf("find_nodes: found %d new nodes", len(newNodes))
	}

	sort.Slice(nodes, func(i, j int) bool {
		di := target.Distance(nodes[i].ID)
		dj := target.Distance(nodes[j].ID)
		return di.Cmp(dj) == -1
	})
	if len(nodes) > A {
		nodes = nodes[:A]
	}

	return nodes, nil
}
