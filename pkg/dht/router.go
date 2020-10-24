package dht

// Router ...
type Router struct {
	// buckets[0] = {1}, 159 common prefix
	// buckets[1] = {2, 3}, 158 common prefix
	// buckets[2] = {4, 5, 6, 7}
	buckets [160]*Bucket
	myID    NodeID
}

// returns -1: if id == self
func (r *Router) bucketIndex(id NodeID) int {
	return r.myID.Distance(id).BitLen() - 1
}
