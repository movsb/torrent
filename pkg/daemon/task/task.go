package task

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/daemon/store"
	"github.com/movsb/torrent/pkg/message"
	"github.com/movsb/torrent/pkg/peer"
	"github.com/movsb/torrent/pkg/torrent"
	trackercommon "github.com/movsb/torrent/pkg/tracker/common"
	trackertcpclient "github.com/movsb/torrent/pkg/tracker/tcp/client"
	trackerudpclient "github.com/movsb/torrent/pkg/tracker/udp/client"
)

// Task ...
type Task struct {
	File     *torrent.File
	InfoHash common.Hash
	BitField *message.BitField
	PM       *store.PieceManager
	clients  map[string]*peer.Peer

	pending chan peer.SinglePieceData
	done    chan peer.SinglePieceData

	mu sync.RWMutex
}

// AddClient ...
func (t *Task) AddClient(client *peer.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if client.PeerAddr == "" {
		panic("no peer address")
	}

	go func() {
		if err := client.Download(t.pending, t.done); err != nil {
			fmt.Printf("download piece failed: %v\n", err)
		}
		t.mu.Lock()
		defer t.mu.Unlock()
		delete(t.clients, client.PeerAddr)
		fmt.Printf("delete client %v\n", client.PeerAddr)
	}()

	t.clients[client.PeerAddr] = client
}

func (t *Task) AnnounceAndDownload() {
	nPieces := t.File.PieceHashes.Count()
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
		t := trackertcpclient.Client{
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
		t := trackerudpclient.Client{
			Address:  trackerURL,
			InfoHash: t.File.InfoHash(),
			MyPeerID: trackercommon.MyPeerID,
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
				fmt.Printf("dial error: %v\n", err)
				return
			}

			closeConn := conn

			defer func() {
				if closeConn != nil {
					closeConn.Close()
				}
			}()

			handshake, err := peer.HandshakeOutgoing(conn, 10, t.File.InfoHash(), trackercommon.MyPeerID)
			if err != nil {
				fmt.Printf("handshake failed: %v\n", err)
				return
			}

			c := peer.Peer{
				HerPeerID:  handshake.PeerID,
				PM:         t.PM,
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
		nPieces := t.File.PieceHashes.Count()
		for donePieces < nPieces {
			piece := <-t.done
			err := t.PM.WritePiece(piece.Index, piece.Data)
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
			percent := float64(donePieces) / float64(t.File.PieceHashes.Count()) * 100
			fmt.Printf("%0.2f piece downloaded: %d / %d | %d / %d\n",
				percent, donePieces, nPieces,
				donePieces*t.File.PieceLength, t.File.Length,
			)
		}
	}()
}
