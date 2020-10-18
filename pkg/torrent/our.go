package torrent

import (
	"github.com/movsb/torrent/pkg/common"
	"github.com/zeebo/bencode"
)

// File ...
type File struct {
	Name     string
	Announce string

	Single bool
	Files  []Item
	Length int64

	PieceLength int
	PieceHashes common.PieceHashes

	rawInfo  bencode.RawMessage
	infoHash common.Hash
}

// InfoHash ...
func (f *File) InfoHash() common.Hash {
	return f.infoHash
}

// Item ...
type Item struct {
	Length int64
	Paths  []string
}
