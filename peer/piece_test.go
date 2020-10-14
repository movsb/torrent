package peer

import (
	"fmt"
	"testing"

	"github.com/movsb/torrent/file"
)

func TestIndex2Files(t *testing.T) {
	files := []file.Item{
		{Length: 80},
		{Length: 140},
		{Length: 50},
		{Length: 130},
	}
	hashes := file.PieceHashes((&[80]byte{})[:])
	ifm := NewIndexFileManager(`test`, false, files, 100, hashes)
	for _, pf := range ifm.piece2files {
		fmt.Printf("%+v\n", pf)
	}
}
