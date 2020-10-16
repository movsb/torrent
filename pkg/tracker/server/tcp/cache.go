package tcptrackerserver

import (
	"fmt"
	"sync"

	tracker "github.com/movsb/torrent/tracker/tcp"
)

type _PeerCache struct {
	mu sync.RWMutex
	m  map[[20]byte]map[string]_PeerCacheEntry
}

func _NewPeerCache() *_PeerCache {
	return &_PeerCache{
		m: make(map[[20]byte]map[string]_PeerCacheEntry),
	}
}

func (c *_PeerCache) Add(ih [20]byte, peerID tracker.PeerID, ip string, port int) []_PeerCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m[ih] == nil {
		c.m[ih] = make(map[string]_PeerCacheEntry)
	}
	c.m[ih][fmt.Sprintf("%s:%d", ip, port)] = _PeerCacheEntry{
		PeerID: peerID,
		IP:     ip,
		Port:   port,
	}
	return c.get(ih)
}

func (c *_PeerCache) Get(ih [20]byte) (peers []_PeerCacheEntry) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.get(ih)
}

func (c *_PeerCache) get(ih [20]byte) (peers []_PeerCacheEntry) {
	for _, p := range c.m[ih] {
		peers = append(peers, p)
	}
	return
}

type _PeerCacheEntry struct {
	PeerID tracker.PeerID
	IP     string
	Port   int
}
