package main

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/movsb/torrent/file"
	"github.com/movsb/torrent/message"
	"github.com/movsb/torrent/peer"
	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/seeder"
	tracker "github.com/movsb/torrent/tracker/tcp"
	udptracker "github.com/movsb/torrent/tracker/udp"
)

// Daemon ...
type Daemon struct {
}

// Task ...
type Task struct {
	File     *file.File
	InfoHash common.InfoHash
	BitField *message.BitField
	IFM      *peer.IndexFileManager
	clients  map[string]*peer.Client

	pending chan peer.SinglePieceData
	done    chan peer.SinglePieceData

	mu sync.RWMutex
}

// AddClient ...
func (t *Task) AddClient(client *peer.Client) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if client.PeerAddr == "" {
		panic("no peer address")
	}

	go func() {
		if err := client.Download(t.pending, t.done); err != nil {
			fmt.Printf("download piece failed: %v", err)
		}
		t.mu.Lock()
		defer t.mu.Unlock()
		delete(t.clients, client.PeerAddr)
		fmt.Printf("delete client %v", client.PeerAddr)
	}()

	t.clients[client.PeerAddr] = client
}

func (t *Task) AnnounceAndDownload() {
	nPieces := t.File.PieceHashes.Len()
	chPieces := make(chan peer.SinglePieceData, nPieces)
	for i := 0; i < nPieces-1; i++ {
		chPieces <- peer.SinglePieceData{
			Index:  i,
			Hash:   t.File.PieceHashes.Index(i),
			Length: t.File.PieceLength,
		}
	}

	lastPieceIndex, lastPieceLength := nPieces-1, t.File.PieceLength
	if remain := int(t.File.Length % int64(t.File.PieceLength)); remain != 0 {
		lastPieceLength = remain
	}
	chPieces <- peer.SinglePieceData{
		Index:  lastPieceIndex,
		Hash:   t.File.PieceHashes.Index(lastPieceIndex),
		Length: lastPieceLength,
	}

	chResult := make(chan peer.SinglePieceData)

	t.pending = chPieces
	t.done = chResult

	trackerURL := t.File.Announce
	if !strings.Contains(trackerURL, "://") {
		trackerURL = "http://" + trackerURL
	}

	u, err := url.Parse(trackerURL)
	if err != nil {
		panic(err)
	}

	var peers []string

	switch u.Scheme {
	case "http", "https":
		t := tracker.TCPTracker{
			Address:  trackerURL,
			InfoHash: t.File.InfoHash(),
		}
		r, err := t.Announce()
		if err != nil {
			panic(err)
		}
		for _, p := range r.Peers {
			peers = append(peers, fmt.Sprintf("%s:%d", p.IP, p.Port))
		}
	case "udp":
		t := udptracker.UDPTracker{
			Address:  trackerURL,
			InfoHash: t.File.InfoHash(),
			MyPeerID: tracker.MyPeerID,
		}
		r, err := t.Announce()
		if err != nil {
			panic(err)
		}
		peers = r.Peers
	}

	for _, p := range peers {
		go func(p string) {
			conn, err := net.DialTimeout("tcp", p, time.Second*10)
			if err != nil {
				fmt.Printf("dial error: %v", err)
				return
			}

			closeConn := conn

			defer func() {
				if closeConn != nil {
					closeConn.Close()
				}
			}()

			handshake, err := peer.HandshakeOutgoing(conn, 10, t.File.InfoHash(), tracker.MyPeerID)
			if err != nil {
				fmt.Printf("handshake failed: %v", err)
				return
			}

			c := peer.Client{
				HerPeerID:  handshake.PeerID,
				Ifm:        t.IFM,
				MyBitField: t.BitField,
				InfoHash:   t.File.InfoHash(),
				PeerAddr:   p,
			}

			c.SetConn(conn)

			if err := c.RecvBitField(); err != nil {
				fmt.Printf("error recv bitbield: %v\n", err)
				return
			}
			if err := c.Send(message.MsgUnChoke, message.UnChoke{}); err != nil {
				fmt.Printf("error send unchoked: %v\n", err)
				return
			}
			if err := c.Send(message.MsgInterested, message.Interested{}); err != nil {
				fmt.Printf("error send unchoked: %v\n", err)
				return
			}

			closeConn = nil

			t.AddClient(&c)
		}(p)
	}
}

func (t *Task) Run() {
	if !t.BitField.AllOnes() {
		t.AnnounceAndDownload()
	}

	go func() {
		donePieces := 0
		nPieces := t.File.PieceHashes.Len()
		for donePieces < nPieces {
			piece := <-t.done
			err := t.IFM.WritePiece(piece.Index, piece.Data)
			if err != nil {
				panic(fmt.Errorf("WritePiece failed: %s", err))
			}

			// TODO(movsb): send this to all peers.
			t.BitField.SetPiece(piece.Index)
			go func(index int) {
				t.mu.RLock()
				defer t.mu.RUnlock()

				for _, client := range t.clients {
					client.HaveCh <- index
				}

			}(piece.Index)

			donePieces++
			percent := float64(donePieces) / float64(t.File.PieceHashes.Len()) * 100
			fmt.Printf("%0.2f piece downloaded: %d / %d | %d / %d\n",
				percent, donePieces, nPieces,
				donePieces*t.File.PieceLength, t.File.Length,
			)
		}
	}()
}

// TaskManager ...
type TaskManager struct {
	mu    sync.RWMutex
	tasks map[common.InfoHash]*Task
}

// AddClient ...
func (t *TaskManager) AddClient(ih common.InfoHash, client *peer.Client) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.tasks[ih].AddClient(client)
}

// LoadTorrent ...
func (t *TaskManager) LoadTorrent(ih common.InfoHash) (*seeder.LoadInfo, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if task, ok := t.tasks[ih]; ok {
		return &seeder.LoadInfo{
			TF:  task.File,
			IFM: task.IFM,
			BF:  task.BitField,
		}, nil
	}

	return nil, fmt.Errorf("no such task")
}

func (t *TaskManager) CreateTask(torrent string, savePath string, bf byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	tf, err := file.ParseFile(torrent)
	if err != nil {
		panic(err)
	}

	if _, ok := t.tasks[tf.InfoHash()]; ok {
		fmt.Printf("task exists")
		return
	}

	task := &Task{
		File:     tf,
		InfoHash: tf.InfoHash(),
		BitField: message.NewBitField(tf.PieceHashes.Len(), bf),
		IFM:      peer.NewIndexFileManager(tf.Name, tf.Single, tf.Files, tf.PieceLength, tf.PieceHashes),

		clients: make(map[string]*peer.Client),
	}

	t.tasks[tf.InfoHash()] = task

	go task.Run()
}

func main() {
	tm := &TaskManager{
		tasks: make(map[common.InfoHash]*Task),
	}

	seeder := seeder.Server{
		Address:     `localhost:8888`,
		MyPeerID:    tracker.MyPeerID,
		LoadTorrent: tm,
	}

	tm.CreateTask("8ce301d28fe97eed1a6ef7feaf296411b375222f.torrent", ".", 0xFF)
	tm.CreateTask("ubuntu.torrent", ".", 0x00)

	if err := seeder.Run(); err != nil {
		panic(err)
	}
}
