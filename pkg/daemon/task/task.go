package task

import (
	"container/list"
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
	busyPeers map[string]*peer.Peer
	idlePeers map[string]*peer.Peer

	pieces *list.List
	done   chan peer.SinglePieceData

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

	t.idlePeers[client.PeerAddr] = client
	log.Printf("add %s to idle", client.PeerAddr)

	client.OnExit = func(p *peer.Peer) {
		t.mu.Lock()
		defer t.mu.Unlock()
		delete(t.busyPeers, p.PeerAddr)
		delete(t.idlePeers, p.PeerAddr)
	}

	go client.Run()
	go client.Poll()

	t.schedule(false)
}

// Run ...
func (t *Task) Run(ctx context.Context) {
	t.initPieces()

	go t.announce(ctx)
	go t.savePiece(ctx)
}

func (t *Task) schedule(lock bool) {
	if lock {
		t.mu.Lock()
		defer t.mu.Unlock()
	}

	var pops []*list.Element

	defer func() {
		log.Printf("moved %d elements to back", len(pops))
		for _, p := range pops {
			t.pieces.MoveToBack(p)
		}
	}()

	for e := t.pieces.Front(); e != nil; e = e.Next() {
		if len(t.idlePeers) <= 0 {
			return
		}
		p := e.Value.(peer.SinglePieceData)
		if t.BitField.HasPiece(p.Index) {
			pops = append(pops, e)
			continue
		}
		busy := make(map[string]*peer.Peer)
		for _, client := range t.idlePeers {
			if _, ok := busy[client.PeerAddr]; ok {
				log.Printf("peer is busy")
				continue
			}
			if !client.HerBitField.HasPiece(p.Index) {
				log.Printf("peer has no index: %d", p.Index)
				continue
			}

			pops = append(pops, e)
			busy[client.PeerAddr] = client

			go func(client *peer.Peer, piece peer.SinglePieceData) {
				if err := client.Download(piece, t.done); err != nil {
					log.Printf("peer download error: %s", err)
					client.OnExit(client)
					return
				}
				t.mu.Lock()
				defer t.mu.Unlock()
				t.idlePeers[client.PeerAddr] = client
				delete(t.busyPeers, client.PeerAddr)
				t.schedule(false)
			}(client, p)
			break
		}
		for a, b := range busy {
			delete(t.idlePeers, a)
			t.busyPeers[a] = b
		}
	}
}
