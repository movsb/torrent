package task

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/movsb/torrent/pkg/peer"
)

func (t *Task) initPieces() {
	nPieces := t.File.PieceHashes.Count()
	chPieces := make(chan peer.SinglePieceData, nPieces)
	for i := 0; i < nPieces-1; i++ {
		chPieces <- peer.SinglePieceData{
			Index:  i,
			Hash:   t.File.PieceHashes.Index(i),
			Length: t.File.PieceLength,
		}
		if i >= 10 {
			break
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
}

func (t *Task) savePiece(ctx context.Context) {
	//donePieces := 0
	donePieces := t.File.PieceHashes.Count() - 10
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

		donePieces++
		percent := float64(donePieces) / float64(t.File.PieceHashes.Count()) * 100
		fmt.Printf("%0.2f piece downloaded, piece: %d / %d, size: %d / %d, speed: %s\n",
			percent, donePieces, nPieces,
			donePieces*t.File.PieceLength, t.File.Length,
			speedString,
		)
	}

	save := func(piece peer.SinglePieceData) {
		if t.BitField.HasPiece(piece.Index) {
			log.Printf("task.savePiece: duplicate piece: %d", piece.Index)
			return
		}
		err := t.PM.WritePiece(piece.Index, piece.Data)
		if err != nil {
			log.Printf("WritePiece failed: %s", err)
			return
		}

		t.BitField.SetPiece(piece.Index)

		go func(index int) {
			t.mu.RLock()
			defer t.mu.RUnlock()

			for _, client := range t.clients {
				select {
				case client.HaveCh <- index:
				default:
					log.Printf("task.savePiece: failed to send have to peer: %s", client.PeerAddr)
				}
			}
		}(piece.Index)

		speed()
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("task.savePiece: context done")
		case piece := <-t.done:
			save(piece)
			if donePieces == nPieces {
				log.Printf("task.savePiece: task done")
				return
			}
		}
	}
}
