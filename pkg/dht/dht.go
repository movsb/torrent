package dht

// Some constants
const (
	K = 20 // The replication factor.
	Α = 3  // α, alpha, not a. The concurency parameter.
)

// DHT ...
type DHT struct {
	router *Router
}

// New ...
func New(myID NodeID) *DHT {
	dht := &DHT{
		router: NewRouter(myID),
	}
	return dht
}
