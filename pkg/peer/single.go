package peer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"sync"

	"github.com/movsb/torrent/pkg/common"
)

// SinglePiece 表示一个分块的数据。
// 用于下载使用。
type SinglePiece struct {
	Index  int         // 分块的索引号
	Hash   common.Hash // 分块的哈希
	Length int         // 分块的长度
	Data   []byte      // 分块数据

	mu sync.Mutex // 保护 Data

	parentCtx    context.Context    // 父级 Context，用于随时退出下载任务
	clientCtx    context.Context    // 用于取消正在下载此分块的 clients
	clientCancel context.CancelFunc // 调用以取消 clients 的下载
}

// NewSinglePiece ...
func NewSinglePiece(ctx context.Context, index int, hash common.Hash, length int) *SinglePiece {
	p := &SinglePiece{
		Index:     index,
		Hash:      hash,
		Length:    length,
		parentCtx: ctx,
	}
	p.clientCtx, p.clientCancel = context.WithCancel(ctx)
	return p
}

// ClientContext 给单个下载 client 的 Context。
func (p *SinglePiece) ClientContext() context.Context {
	return p.clientCtx
}

// SetData 用于某个 client 下载完后调用。
// 一个分块可能同时被多个 clients 下载，但是只有第一个
// 下载完成且校验通过的才算成功。
// 一旦完成，其它的下载都会被取消。
func (p *SinglePiece) SetData(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 已经下载完成了，正在退出
	if p.Data != nil {
		return nil
	}

	if err := p.setData(data); err != nil {
		return err
	}

	// 下载成功了。
	p.clientCancel()

	return nil
}

func (p *SinglePiece) setData(data []byte) error {
	if len(data) != p.Length {
		return fmt.Errorf(`data length not match, drop data`)
	}
	got := sha1.Sum(data)
	if !bytes.Equal(p.Hash[:], got[:]) {
		return fmt.Errorf("check integrity failed, drop data")
	}
	p.Data = make([]byte, p.Length)
	copy(p.Data, data)
	return nil
}

type _SinglePiecePartial struct {
	sp         *SinglePiece
	done       chan error
	data       []byte
	downloaded int
	requested  int
	backlog    int
}
