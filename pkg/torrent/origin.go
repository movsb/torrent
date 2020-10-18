package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math"

	"github.com/movsb/torrent/pkg/common"
	"github.com/zeebo/bencode"
)

// _File ...
type _File struct {
	Announce string             `bencode:"announce"`
	Info     bencode.RawMessage `bencode:"info"`
}

func (f *_File) convert() (*File, error) {
	i := _Info{}
	if err := bencode.DecodeBytes(f.Info, &i); err != nil {
		return nil, err
	}

	if i.NameUTF8 != `` {
		i.Name = i.NameUTF8
	}

	c := &File{
		Announce:    f.Announce,
		Name:        i.Name,
		Length:      i.Length,
		PieceLength: i.PieceLength,
		Files:       make([]Item, 0, len(i.Files)),

		rawInfo:  f.Info,
		infoHash: f.infoHash(),
	}

	if len(i.Pieces)%sha1.Size != 0 {
		return nil, fmt.Errorf(`invalid hash from pieces: len=%d`, len(i.Pieces))
	}

	// if it is a single file torrent,
	// transform to multiple files torrent.
	if i.Length > 0 {
		c.Single = true
		i.Files = append(i.Files, _Item{
			Length: i.Length,
			Paths:  []string{i.Name},
		})
	} else {
		c.Length = 0
		for _, item := range i.Files {
			c.Length += item.Length
		}
	}

	nPieces := len(i.Pieces) / sha1.Size
	calcNumPieces := int(math.Ceil(float64(c.Length) / float64(i.PieceLength)))
	if calcNumPieces != nPieces {
		return nil, fmt.Errorf(`invalid hash from pieces: calcNumPieces mismatch`)
	}
	c.PieceHashes = common.PieceHashes(i.Pieces)

	for _, item := range i.Files {
		it := Item{
			Length: item.Length,
			Paths:  item.Paths,
		}
		if len(item.PathsUTF8) > 0 {
			it.Paths = item.PathsUTF8
		}
		c.Files = append(c.Files, it)
	}

	return c, nil
}

func (f *_File) infoHash() [20]byte {
	buf := bytes.Buffer{}
	if err := bencode.NewEncoder(&buf).Encode(f.Info); err != nil {
		panic(err)
	}
	return sha1.Sum(buf.Bytes())
}

// _Info ...
type _Info struct {
	Name        string  `bencode:"name"`
	NameUTF8    string  `bencode:"name.utf-8"`
	Files       []_Item `bencode:"files"`
	Length      int64   `bencode:"length,omitempty"`
	PieceLength int     `bencode:"piece length"`
	Pieces      []byte  `bencode:"pieces"`
}

// _Item ...
type _Item struct {
	Length    int64    `bencode:"length"`
	Paths     []string `bencode:"path"`
	PathsUTF8 []string `bencode:"path.utf-8"`
}
