package store

import (
	"fmt"
	"testing"

	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/torrent"
)

func TestIndex2Files(t *testing.T) {
	file := torrent.File{
		Files: []torrent.Item{
			{Length: 80},
			{Length: 140},
			{Length: 50},
			{Length: 130},
		},
		PieceLength: 100,
		PieceHashes: common.PieceHashes((&[80]byte{})[:]),
	}
	pm := NewPieceManager(&file)
	for _, pf := range pm.piece2files {
		fmt.Printf("%+v\n", pf)
	}
}
