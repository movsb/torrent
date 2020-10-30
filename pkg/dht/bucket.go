package dht

import (
	"container/list"
	"log"
	"time"
)

type _NodeSpec struct {
	lastUpdated time.Time
	node        Node
}

func (n _NodeSpec) good() bool {
	return time.Since(n.lastUpdated) < time.Minute*15
}

// Bucket ...
type Bucket struct {
	// Least-recently seen node at the head,
	// most-recently seen node at the tail.
	nodes *list.List
}

// Add ...
func (b *Bucket) Add(node Node) {
	// already exists, move to back.
	for iter := b.nodes.Front(); iter != nil; iter = iter.Next() {
		spec := iter.Value.(*_NodeSpec)
		if spec.node.ID == node.ID {
			spec.lastUpdated = time.Now()
			b.nodes.MoveToBack(iter)
			log.Printf("Bucket: move to back: %v", node.ID)
			return
		}
	}

	// not already exists

	// not enough nodes
	if b.nodes.Len() < K {
		b.nodes.PushBack(&_NodeSpec{
			lastUpdated: time.Now(),
			node:        node,
		})
		log.Printf("Bucket: push back: %v", node.ID)
		return
	}

	// TODO(movsb): ping to see if it exists
	// if least-recently seen node not responded, evict it. And the new node is inserted into the tail.
}

// Nodes ...
func (b *Bucket) Nodes() []Node {
	nodes := make([]Node, 0, b.nodes.Len())
	for e := b.nodes.Front(); e != nil; e = e.Next() {
		spec := e.Value.(*_NodeSpec)
		nodes = append(nodes, spec.node)
	}
	return nodes
}
