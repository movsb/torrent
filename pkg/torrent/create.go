package torrent

import (
	"crypto/sha1"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeebo/bencode"
)

// Creator is the torrent file creator.
type Creator struct {
	f    _CreateFile
	path string
}

type _CreateFile struct {
	Announce string `bencode:"announce,omitempty"`
	Info     _Info  `bencode:"info,omitempty"`
}

// NewCreator ...
func NewCreator(path string) *Creator {
	return &Creator{
		path: path,
	}
}

// Create ...
func (c *Creator) Create(w io.Writer) error {
	stat, err := os.Stat(c.path) // Lstat?
	if err != nil {
		return fmt.Errorf("creator: stat failed: %v", err)
	}

	var (
		prefix string
		size   int64
		files  []_Item
	)

	switch {
	default:
		return fmt.Errorf("creator: invalid file type: %v", stat.Mode())
	case stat.IsDir():
		prefix, size, files, err = c.createDir()
	case stat.Mode().IsRegular():
		prefix, size, files, err = c.createFile()
	}

	if err != nil {
		return fmt.Errorf("creator: %v", err)
	}

	if err := c.calcPieceHashes(prefix, size, files); err != nil {
		return fmt.Errorf("creator: calc piece hashes failed: %v", err)
	}

	return bencode.NewEncoder(w).Encode(c.f)
}

func (c *Creator) createFile() (string, int64, []_Item, error) {
	stat, err := os.Stat(c.path)
	if err != nil {
		return ``, 0, nil, err
	}

	c.f.Info.Name = stat.Name()
	fileSize := stat.Size()
	c.f.Info.Length = fileSize

	pieceLength, pieceCount := calcPiece(fileSize)
	c.f.Info.Pieces = make([]byte, 20*pieceCount)
	c.f.Info.PieceLength = pieceLength

	abs, err := filepath.Abs(c.path)
	if err != nil {
		return ``, 0, nil, err
	}

	dir := filepath.Dir(abs)
	files := []_Item{{
		Length: fileSize,
		Paths:  []string{filepath.Base(abs)},
	}}

	return dir, fileSize, files, nil
}

func (c *Creator) createDir() (string, int64, []_Item, error) {
	prefix, length, files, err := fileList(c.path)
	if err != nil {
		return ``, 0, nil, err
	}

	c.f.Info.Name = filepath.Base(prefix)
	c.f.Info.Length = 0

	pieceLength, pieceCount := calcPiece(length)
	c.f.Info.Pieces = make([]byte, 20*pieceCount)
	c.f.Info.PieceLength = pieceLength

	c.f.Info.Files = files

	return prefix, length, files, nil
}

func (c *Creator) calcPieceHashes(prefix string, size int64, files []_Item) error {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		path := filepath.Join(prefix, filepath.Join(file.Paths...))
		paths = append(paths, path)
	}

	piece := make([]byte, c.f.Info.PieceLength)
	mr := NewMultiReader(paths)

	pieceCount := len(c.f.Info.Pieces) / 20
	pieceLength := c.f.Info.PieceLength

	for i := 0; i < pieceCount; i++ {
		if i == pieceCount-1 {
			if remain := size % int64(pieceLength); remain != 0 {
				pieceLength = int(remain)
			}
		}
		_, err := io.ReadFull(mr, piece[0:pieceLength])
		if err != nil {
			return err
		}
		sum := sha1.Sum(piece[0:pieceLength])
		copy(c.f.Info.Pieces[i*20:i*20+20], sum[:])
	}

	if n, err := mr.Read(piece); n > 0 || err != io.EOF {
		return fmt.Errorf("file size changed")
	}

	return nil
}

func calcPiece(totalSize int64) (pieceLength int, pieceCount int) {
	const (
		maxPieceCount  int = 20000
		minPieceLength int = 256 << 10
	)
	var (
		pieceLength64 = int64(minPieceLength)
	)

	for {
		pieceCount64 := totalSize / pieceLength64
		if totalSize%pieceLength64 != 0 {
			pieceCount64++
		}
		if pieceCount64 <= int64(maxPieceCount) {
			if pieceLength64 < math.MaxInt32 && pieceCount64 < math.MaxInt32 {
				return int(pieceLength64), int(pieceCount64)
			}
		}
		pieceLength64 *= 2
	}
}

func fileList(dir string) (string, int64, []_Item, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return ``, 0, nil, err
	}
	if len(dir) == 1 {
		return ``, 0, nil, fmt.Errorf(`dir cannot be root`)
	}

	var size int64
	var files []_Item

	if err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}

			item := _Item{
				Length: info.Size(),
				Paths:  strings.Split(rel, string(os.PathSeparator)),
			}

			// TODO(movsb): max file count?
			files = append(files, item)

			size += item.Length

			return nil
		},
	); err != nil {
		return ``, 0, nil, err
	}

	return dir, size, files, nil
}

// MultiReader is a modified version of io.MultiReader that
// supports opening files when needed.
// Not thread-safe.
type MultiReader struct {
	files []string
	fps   []*os.File
}

// NewMultiReader ...
func NewMultiReader(files []string) *MultiReader {
	r := &MultiReader{
		files: files,
		fps:   make([]*os.File, len(files)),
	}
	return r
}

func (mr *MultiReader) Read(p []byte) (n int, err error) {
	for len(mr.fps) > 0 {
		if mr.fps[0] == nil {
			fp, err := os.Open(mr.files[0])
			if err != nil {
				return 0, err
			}
			mr.fps[0] = fp
		}
		n, err = mr.fps[0].Read(p)
		if err == io.EOF {
			mr.fps[0].Close()
			mr.fps = mr.fps[1:]
			mr.files = mr.files[1:]
		}
		if n > 0 || err != io.EOF {
			if err != nil && err != io.EOF {
				mr.fps[0].Close()
			}
			if err == io.EOF && len(mr.fps) > 0 {
				// Don't return EOF yet. More readers remain.
				err = nil
			}
			return
		}
	}
	return 0, io.EOF
}
