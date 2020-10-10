package file

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/zeebo/bencode"
)

// _File ...
type _File struct {
	Announce     string             `bencode:"announce"`
	Comment      string             `bencode:"comment"`
	CommentUTF8  string             `bencode:"comment.utf-8"`
	CreationDate int                `bencode:"creation_date"`
	Info         bencode.RawMessage `bencode:"info"`
}

func (f *_File) convert() (*File, error) {
	i := _Info{}
	if err := bencode.DecodeBytes(f.Info, &i); err != nil {
		return nil, err
	}

	c := &File{
		Announce:     f.Announce,
		Comment:      f.Comment,
		CreationDate: f.CreationDate,
		Info: Info{
			Name:        i.Name,
			Length:      i.Length,
			PieceLength: i.PieceLength,
			Pieces:      i.Pieces,
			Files:       make([]Item, 0, len(i.Files)),
		},

		rawInfo:  f.Info,
		infoHash: f.infoHash(),
	}

	if f.CommentUTF8 != "" {
		c.Comment = f.CommentUTF8
	}
	if i.NameUTF8 != `` {
		c.Info.Name = i.NameUTF8
	}

	for _, item := range i.Files {
		it := Item{
			Length: item.Length,
			Paths:  item.Paths,
		}
		if len(item.PathsUTF8) > 0 {
			it.Paths = item.PathsUTF8
		}
		c.Info.Files = append(c.Info.Files, it)
	}

	return c, nil
}

func (f *_File) infoHash() string {
	buf := bytes.Buffer{}
	if err := bencode.NewEncoder(&buf).Encode(f.Info); err != nil {
		panic(err)
	}
	sum := sha1.Sum(buf.Bytes())
	return fmt.Sprintf("%x", sum)
}

// _Info ...
type _Info struct {
	Name        string  `bencode:"name"`
	NameUTF8    string  `bencode:"name.utf-8"`
	Files       []_Item `bencode:"files"`
	Length      int     `bencode:"length,omitempty"`
	PieceLength int     `bencode:"piece length"`
	Pieces      []byte  `bencode:"pieces"`
}

// _Item ...
type _Item struct {
	Length    int      `bencode:"length"`
	Paths     []string `bencode:"path"`
	PathsUTF8 []string `bencode:"path.utf-8"`
}

// File ...
type File struct {
	Announce     string `bencode:"announce"`
	Comment      string `bencode:"comment"`
	CreationDate int    `bencode:"creation_date"`
	Info         Info   `bencode:"info"`

	rawInfo  bencode.RawMessage
	infoHash string
}

// InfoHash ...
func (f *File) InfoHash() string {
	return f.infoHash
}

// Info ...
type Info struct {
	Name        string `bencode:"name"`
	Files       []Item `bencode:"files"`
	Length      int    `bencode:"length,omitempty"`
	PieceLength int    `bencode:"piece length"`
	Pieces      []byte `bencode:"pieces"`
}

// Item ...
type Item struct {
	Length int      `bencode:"length"`
	Paths  []string `bencode:"path"`
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
