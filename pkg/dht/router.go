package dht

import (
	"log"
)

// Router ...
type Router struct {
	// buckets[0] = {1}, 159 common prefix
	// buckets[1] = {2, 3}, 158 common prefix
	// buckets[2] = {4, 5, 6, 7}
	buckets [160]*Bucket
	myID    NodeID
}

// NewRouter ...
func NewRouter(myID NodeID) *Router {
	r := &Router{
		myID: myID,
	}
	for i := 0; i < len(r.buckets); i++ {
		r.buckets[i] = new(Bucket)
	}
	return r
}

func (r *Router) addExtraBootstrapNodes(addrs []string) {

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
func (r *Router) Upsert(node Node) {
	bucket := r.bucket(node.ID)
	if bucket == nil {
		log.Printf("router: trying to add self as node")
		return
	}
	bucket.Add(node)
}
