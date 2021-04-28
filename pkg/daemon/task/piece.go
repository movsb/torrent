package task

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/movsb/torrent/pkg/peer"
)

func (t *Task) initPieces() {
	list := list.New()
	nPieces := t.File.PieceHashes.Count()
	remain := int(t.File.Length % int64(t.File.PieceLength))

	for i := 0; i < nPieces; i++ {
		length := t.File.PieceLength
		if i == nPieces-1 && remain != 0 {
			length = remain
		}

		piece := peer.NewSinglePiece(context.TODO(), i, t.File.PieceHashes.Index(i), length)
		list.PushBack(piece)

		//if i >= 10 {
		//	break
		//}
	}

	t.pieces = list
	t.gotPiece = make(chan *peer.SinglePiece, 100)
}

func (t *Task) savePiece(ctx context.Context) {
	donePieces := 0
	nPieces := t.File.PieceHashes.Count()

	lastTime := time.Now()
	lastSpeed := float64(0)
	lastDonePieces := 0

	speed := func() {
		now := time.Now()
		if sub := now.Sub(lastTime); sub >= time.Millisecond*500 {
			n := donePieces - lastDonePieces
			lastSpeed = float64(n*t.File.PieceLength) / sub.Seconds()
			lastDonePieces = donePieces
			lastTime = now
		}

		var speedString string
		if lastSpeed >= 1<<20 {
			speedString = fmt.Sprintf("%.2fM/s", lastSpeed/(1<<20))
		} else if lastSpeed >= 1<<10 {
			speedString = fmt.Sprintf("%.2fK/s", lastSpeed/(1<<10))
		} else {
			speedString = fmt.Sprintf("%.2fB/s", lastSpeed/(1<<0))
		}

		t.mu.RLock()
		defer t.mu.RUnlock()

		donePieces++
		percent := float64(donePieces) / float64(t.File.PieceHashes.Count()) * 100
		fmt.Printf("%0.2f piece downloaded, piece: %d / %d, size: %d / %d, speed: %s, idle: %d, busy: %d\n",
			percent, donePieces, nPieces,
			donePieces*t.File.PieceLength, t.File.Length,
			speedString,
			len(t.idlePeers), len(t.busyPeers),
		)
	}

	save := func(piece *peer.SinglePiece) bool {
		if t.BitField.HasPiece(piece.Index) {
			log.Printf("task.savePiece: duplicate piece: %d", piece.Index)
			return false
		}
		err := t.PM.WritePiece(piece.Index, piece.Data)
		if err != nil {
			log.Printf("WritePiece failed: %s", err)
			return false
		}

		t.BitField.SetPiece(piece.Index)

		go func(index int) {
			t.mu.RLock()
			defer t.mu.RUnlock()

			for _, client := range t.busyPeers {
				select {
				case client.HaveCh <- index:
				default:
					log.Printf("task.savePiece: failed to send have to peer: %s", client.PeerAddr)
				}
			}

			for _, client := range t.idlePeers {
				select {
				case client.HaveCh <- index:
				default:
					log.Printf("task.savePiece: failed to send have to peer: %s", client.PeerAddr)
				}
			}
		}(piece.Index)

		speed()
		return true
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("task.savePiece: context done")
		case piece := <-t.gotPiece:
			if save(piece) && donePieces == nPieces {
				log.Printf("task.savePiece: task done")
				return
			}
		}
	}
}
