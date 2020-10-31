package dht

import (
	"log"
	"sort"
	"sync"
)

// Router ...
type Router struct {
	// buckets[0] = {1}, 159 common prefix
	// buckets[1] = {2, 3}, 158 common prefix
	// buckets[2] = {4, 5, 6, 7}
	buckets [160]*Bucket
	myID    NodeID
	mu      sync.Mutex
}

// NewRouter ...
func NewRouter(myID NodeID) *Router {
	r := &Router{
		myID: myID,
	}
	for i := 0; i < len(r.buckets); i++ {
		r.buckets[i] = newBucket()
	}
	return r
}

// returns -1: if id == self
func (r *Router) bucketIndex(id NodeID) int {
	return r.myID.Distance(id).BitLen() - 1
}

func (r *Router) bucket(id NodeID) *Bucket {
	index := r.bucketIndex(id)
	if index == -1 {
		return nil
	}
	return r.buckets[index]
}

// Upsert updates or inserts a node.
func (r *Router) Upsert(node Node, ping bool) {
	bucket := r.bucket(node.ID)
	if bucket == nil {
		log.Printf("router: trying to add self as node")
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	bucket.Add(node)
}

// Closest ...
func (r *Router) Closest(id NodeID, k int) []Node {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodes := make([]Node, 0, k)
	bucketIndex := r.bucketIndex(id)
	if bucketIndex == -1 {
		// TODO add myself
		// nodes = append(nodes, r.myID)
		bucketIndex = 0
	}
	for i := bucketIndex; i >= 0; i-- {
		if len(nodes) >= k {
			break
		}
		nodes = append(nodes, r.buckets[i].Nodes()...)
	}
	for i := bucketIndex + 1; i < len(r.buckets); i++ {
		if len(nodes) >= k {
			break
		}
		nodes = append(nodes, r.buckets[i].Nodes()...)
	}
	sort.Slice(nodes, func(i, j int) bool {
		di := id.Distance(nodes[i].ID)
		dj := id.Distance(nodes[j].ID)
		return di.Cmp(dj) == -1
	})
	if len(nodes) > k {
		nodes = nodes[:k]
	}
	return nodes
}
