package task

import (
	"fmt"
	"sync"

	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/daemon/store"
	"github.com/movsb/torrent/pkg/message"
	"github.com/movsb/torrent/pkg/peer"
	"github.com/movsb/torrent/pkg/seeder"
	"github.com/movsb/torrent/pkg/torrent"
)

// Manager ...
type Manager struct {
	mu    sync.RWMutex
	tasks map[common.Hash]*Task
}

// NewManager ...
func NewManager() *Manager {
	m := &Manager{
		tasks: make(map[common.Hash]*Task),
	}
	return m
}

// AddClient ...
func (t *Manager) AddClient(ih common.Hash, client *peer.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.tasks[ih].AddClient(client)
}

// LoadTorrent ...
func (t *Manager) LoadTorrent(ih common.Hash) (*seeder.LoadInfo, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if task, ok := t.tasks[ih]; ok {
		return &seeder.LoadInfo{
			TF: task.File,
			PM: task.PM,
			BF: task.BitField,
		}, nil
	}

	return nil, fmt.Errorf("no such task")
}

func (t *Manager) CreateTask(file string, savePath string, bf byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	tf, err := torrent.ParseFile(file)
	if err != nil {
		panic(err)
	}

	if _, ok := t.tasks[tf.InfoHash()]; ok {
		fmt.Printf("task exists\n")
		return
	}

	task := &Task{
		File:     tf,
		InfoHash: tf.InfoHash(),
		BitField: message.NewBitField(tf.PieceHashes.Count(), bf),
		PM:       store.NewPieceManager(tf),

		clients: make(map[string]*peer.Peer),
	}

	t.tasks[tf.InfoHash()] = task

	go task.Run()
}
