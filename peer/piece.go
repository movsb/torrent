package peer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/movsb/torrent/file"
)

// SinglePieceData ...
type SinglePieceData struct {
	Index  int
	Hash   []byte
	Length int
	Data   []byte
}

type _IndexedFile struct {
	fd     int
	offset int64
	length int
}

// IndexFileManager ...
type IndexFileManager struct {
	mu sync.RWMutex

	name        string
	single      bool
	pieceLength int

	files []file.Item
	fds   []*os.File

	hashes      file.PieceHashes
	piece2files [][]_IndexedFile
}

// NewIndexFileManager ...
func NewIndexFileManager(name string, single bool, files []file.Item, pieceLength int, hashes file.PieceHashes) *IndexFileManager {
	ifm := &IndexFileManager{
		name:        name,
		single:      single,
		pieceLength: pieceLength,
		files:       files,
		hashes:      hashes,
	}

	ifm.fds = make([]*os.File, len(files))
	ifm.piece2files = make([][]_IndexedFile, hashes.Len())

	var (
		pieceIndex    int
		pieceOffset   int
		pieceLength64 int64
	)

	indexFile := func(pi, fi int, fo int64, fl int) {
		ifm.piece2files[pi] = append(ifm.piece2files[pi],
			_IndexedFile{
				fd:     fi,
				offset: fo,
				length: fl,
			},
		)
	}

	pieceLength64 = int64(pieceLength)
	for fi, file := range files {
		for fileOffset := int64(0); fileOffset < file.Length; {
			fileRemain := file.Length - fileOffset
			pieceRemain := pieceLength64 - int64(pieceOffset)
			switch diff := fileRemain - pieceRemain; {
			case diff >= 0:
				indexFile(pieceIndex, fi, fileOffset, int(pieceRemain))
				fileOffset += pieceRemain
				pieceOffset = 0
				pieceIndex++
				pieceLength64 = int64(pieceLength)
			case diff < 0:
				indexFile(pieceIndex, fi, fileOffset, int(fileRemain))
				fileOffset += fileRemain
				pieceOffset += int(fileRemain)
			}
		}
	}

	return ifm
}

// Close ...
func (p *IndexFileManager) Close() error {
	var lastErr error
	for _, f := range p.fds {
		if err := f.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// PieceCount ...
func (p *IndexFileManager) PieceCount() int {
	return p.hashes.Len()
}

// ReadPiece ...
// TODO merge read & write.
func (p *IndexFileManager) ReadPiece(index int) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.openFiles(index); err != nil {
		return nil, fmt.Errorf("IndexFileManager.ReadPiece failed: %v", err)
	}

	offset := 0
	files := p.piece2files[index]
	data := make([]byte, p.pieceLength)

	for _, f := range files {
		fp := p.fds[f.fd]
		block := data[offset : offset+f.length]
		_, err := fp.ReadAt(block, f.offset)
		if err != nil {
			return nil, fmt.Errorf("IndexFileManager.ReadPiece failed: %v", err)
		}
		offset += f.length
	}

	// if offset != len(data) {
	// 	return nil, fmt.Errorf("IndexFileManager.ReadPiece: offset != len(data)")
	// }

	return data, nil
}

// WritePiece ...
func (p *IndexFileManager) WritePiece(index int, data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.openFiles(index); err != nil {
		return fmt.Errorf("IndexFileManager.WritePiece failed: %v", err)
	}

	offset := 0
	files := p.piece2files[index]

	for _, f := range files {
		fp := p.fds[f.fd]
		block := data[offset : offset+f.length]
		_, err := fp.WriteAt(block, f.offset)
		if err != nil {
			return fmt.Errorf("IndexFileManager.WritePiece failed: %v", err)
		}
		offset += f.length
	}

	if offset != len(data) {
		return fmt.Errorf("IndexFileManager.WritePiece: offset != len(data)")
	}

	return nil
}

func (p *IndexFileManager) openFiles(index int) error {
	if index < 0 || index >= len(p.piece2files) {
		return fmt.Errorf("IndexFileManager.openFiles: invalid index %d", index)
	}
	files := p.piece2files[index]
	for _, file := range files {
		if fd := p.fds[file.fd]; fd == nil {
			segments := p.files[file.fd].Paths
			dir, name := `.`, segments[len(segments)-1]
			if !p.single || len(segments) > 1 {
				dir = filepath.Join(segments[0 : len(segments)-1]...)
				dir = filepath.Join(p.name, dir)
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("IndexFileManager.openFiles: os.MkdirAll failed: %v", err)
				}
			}
			fullPath := filepath.Join(dir, name)
			fp, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return fmt.Errorf("IndexFileManager.openFiles: os.OpenFile failed: %v", err)
			}
			p.fds[file.fd] = fp
		}
	}
	return nil
}
