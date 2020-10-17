package file

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math"
	"os"

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
	c.PieceHashes = PieceHashes(i.Pieces)

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

// File ...
type File struct {
	Name     string
	Announce string

	Single bool
	Files  []Item
	Length int64

	PieceLength int
	PieceHashes PieceHashes

	rawInfo  bencode.RawMessage
	infoHash common.InfoHash
}

// InfoHash ...
func (f *File) InfoHash() common.InfoHash {
	return f.infoHash
}

// Item ...
type Item struct {
	Length int64
	Paths  []string
}

// Hash ...
type Hash [sha1.Size]byte

func (h Hash) String() string {
	return fmt.Sprintf("%x", [sha1.Size]byte(h))
}

// PieceHashes ...
type PieceHashes []byte

// Len ...
func (p PieceHashes) Len() int {
	return len(p) / sha1.Size
}

// Index ...
func (p PieceHashes) Index(index int) []byte {
	s := sha1.Size * index
	return p[s : s+sha1.Size]
}

// MarshalYAML ...
func (p PieceHashes) MarshalYAML() (interface{}, error) {
	list := make([]string, p.Len())
	for i := 0; i < p.Len(); i++ {
		list[i] = fmt.Sprintf("%x", p.Index(i))
	}
	return list, nil
}

func ParseFile(path string) (*File, error) {
	fp, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	f := _File{}
	if err := bencode.NewDecoder(fp).Decode(&f); err != nil {
		panic(err)
	}

	return f.convert()
}

func ParseFileToInterface(path string) (interface{}, error) {
	fp, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	var f interface{}
	if err := bencode.NewDecoder(fp).Decode(&f); err != nil {
		panic(err)
	}
	return f, nil
}
