package task

import (
	"context"
	"log"
	"sync"

	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/daemon/store"
	"github.com/movsb/torrent/pkg/message"
	"github.com/movsb/torrent/pkg/peer"
	"github.com/movsb/torrent/pkg/torrent"
)

// Task ...
type Task struct {
	File     *torrent.File
	InfoHash common.Hash
	BitField *message.BitField
	PM       *store.PieceManager

	// map from peer address to peer.
	clients map[string]*peer.Peer

	pending chan peer.SinglePieceData
	done    chan peer.SinglePieceData

	mu sync.RWMutex
}

// AddClient ...
func (t *Task) AddClient(client *peer.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if client.PeerAddr == "" {
		log.Printf("no peer address")
		return
	}

	t.clients[client.PeerAddr] = client

	go func() {
		if err := client.Download(t.pending, t.done); err != nil {
			log.Printf("download piece failed: %v\n", err)
		}
		t.mu.Lock()
		defer t.mu.Unlock()
		delete(t.clients, client.PeerAddr)
	}()
}

// Run ...
func (t *Task) Run(ctx context.Context) {
	t.initPieces()

	go t.announce(ctx)
	go t.savePiece(ctx)
}
