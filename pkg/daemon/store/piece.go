package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/movsb/torrent/file"
)

type _IndexedFile struct {
	index  int
	offset int64
	length int
}

// PieceManager ...
type PieceManager struct {
	mu sync.RWMutex

	// The parsed torrent file.
	f *file.File

	// File handles for each file in the torrent.
	// Opens on demand.
	fds []*os.File

	// A piece may span multiple files.
	piece2files [][]_IndexedFile
}

// NewPieceManager ...
func NewPieceManager(f *file.File) *PieceManager {
	pm := &PieceManager{
		f:           f,
		fds:         make([]*os.File, len(f.Files)),
		piece2files: make([][]_IndexedFile, f.PieceHashes.Len()),
	}

	pm.calcFiles()

	return pm
}

func (p *PieceManager) calcFiles() {
	var (
		pieceIndex    int
		pieceOffset   int
		pieceLength64 int64
	)

	indexFile := func(pieceIndex, fileIndex int, fileOffset int64, fileLength int) {
		p.piece2files[pieceIndex] = append(
			p.piece2files[pieceIndex],
			_IndexedFile{
				index:  fileIndex,
				offset: fileOffset,
				length: fileLength,
			},
		)
	}

	pieceLength64 = int64(p.f.PieceLength)
	for fi, file := range p.f.Files {
		for fileOffset := int64(0); fileOffset < file.Length; {
			fileRemain := file.Length - fileOffset
			pieceRemain := pieceLength64 - int64(pieceOffset)
			switch diff := fileRemain - pieceRemain; {
			case diff >= 0:
				indexFile(pieceIndex, fi, fileOffset, int(pieceRemain))
				fileOffset += pieceRemain
				pieceOffset = 0
				pieceIndex++
				pieceLength64 = int64(p.f.PieceLength)
			case diff < 0:
				indexFile(pieceIndex, fi, fileOffset, int(fileRemain))
				fileOffset += fileRemain
				pieceOffset += int(fileRemain)
			}
		}
	}
}

// Close ...
func (p *PieceManager) Close() error {
	var lastErr error
	for _, f := range p.fds {
		if err := f.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// PieceCount ...
func (p *PieceManager) PieceCount() int {
	return p.f.PieceHashes.Len()
}

// ReadPiece ...
// TODO merge read & write.
func (p *PieceManager) ReadPiece(index int) ([]byte, error) {
	if err := p.openFiles(index); err != nil {
		return nil, fmt.Errorf("PieceManager.ReadPiece failed: %v", err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	offset := 0
	files := p.piece2files[index]
	data := make([]byte, p.f.PieceLength)

	for _, f := range files {
		fp := p.fds[f.index]
		block := data[offset : offset+f.length]
		_, err := fp.ReadAt(block, f.offset)
		if err != nil {
			return nil, fmt.Errorf("PieceManager.ReadPiece failed: %v", err)
		}
		offset += f.length
	}

	// TODO(movsb): check this
	// if offset != len(data) {
	// 	return nil, fmt.Errorf("PieceManager.ReadPiece: offset != len(data)")
	// }

	return data, nil
}

// WritePiece ...
func (p *PieceManager) WritePiece(index int, data []byte) error {
	if err := p.openFiles(index); err != nil {
		return fmt.Errorf("PieceManager.WritePiece failed: %v", err)
	}

	// There won't be two writes for one piece index,
	// So it is ok to just Read-Lock?
	p.mu.RLock()
	defer p.mu.RUnlock()

	offset := 0
	files := p.piece2files[index]

	for _, f := range files {
		fp := p.fds[f.index]
		block := data[offset : offset+f.length]
		_, err := fp.WriteAt(block, f.offset)
		if err != nil {
			return fmt.Errorf("PieceManager.WritePiece failed: %v", err)
		}
		offset += f.length
	}

	if offset != len(data) {
		return fmt.Errorf("PieceManager.WritePiece: offset != len(data)")
	}

	return nil
}

// TODO(movsb): on read, this will also open the file, which causes an
// empty file to be created, if it doesn't exist.
func (p *PieceManager) openFiles(index int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index < 0 || index >= len(p.piece2files) {
		return fmt.Errorf("PieceManager.openFiles: invalid index %d", index)
	}

	for _, file := range p.piece2files[index] {
		// if this file is already open, does nothing.
		if fd := p.fds[file.index]; fd != nil {
			continue
		}

		segments := p.f.Files[file.index].Paths
		dir, name := `.`, segments[len(segments)-1]

		// create those parent directories first.
		if !p.f.Single || len(segments) > 1 {
			dir = filepath.Join(segments[0 : len(segments)-1]...)
			dir = filepath.Join(p.f.Name, dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("PieceManager.openFiles: os.MkdirAll failed: %v", err)
			}
		}

		fullPath := filepath.Join(dir, name)
		fp, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("PieceManager.openFiles: os.OpenFile failed: %v", err)
		}

		p.fds[file.index] = fp
	}
	return nil
}
