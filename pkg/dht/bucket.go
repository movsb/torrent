package dht

import (
	"container/list"
	"log"
)

// Bucket ...
type Bucket struct {
	nodes *list.List
}

// Update ...
func (b *Bucket) Update(node *Node) {
	// already exists, move to back.
	for iter := b.nodes.Front(); iter != nil; iter = iter.Next() {
		if iter.Value.(*Node).ID == node.ID {
			b.nodes.MoveToBack(iter)
			log.Printf("Bucket: move to back: %v", node.ID)
			return
		}
	}

	// not already exists

	// not enough nodes
	if b.nodes.Len() < K {
		b.nodes.PushBack(node)
		log.Printf("Bucket: push back: %v", node.ID)
		return
	}

	// TODO(movsb): ping to see if it exists
}
