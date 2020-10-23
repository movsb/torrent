package task

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/movsb/torrent/pkg/message"
	"github.com/movsb/torrent/pkg/peer"
	trackercommon "github.com/movsb/torrent/pkg/tracker/common"
	trackertcpclient "github.com/movsb/torrent/pkg/tracker/tcp/client"
	trackerudpclient "github.com/movsb/torrent/pkg/tracker/udp/client"
)

type _Announce struct {
	address  string
	interval int
}

// announce ...
func (t *Task) announce(ctx context.Context) error {
	log.Printf("task.announce-ing\n")

	execute := func() {
		interval, peers, err := t.announceOne(ctx, t.File.Announce)
		if err != nil {
			log.Printf("task.announce: announce failed: %v", err)
			return
		}
		if interval <= 0 {
			log.Printf("task.announce: interval <= 0")
			return
		}
		_ = interval
		t.spawnPeers(ctx, peers)
	}

	execute()

	ticker := time.NewTicker(time.Minute * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("task.announce: context done")
			return nil
		case <-ticker.C:
			execute()
		}
	}
}

func (t *Task) spawnPeers(ctx context.Context, peers []string) {
	log.Printf("task.spawnPeers entering")

	t.mu.Lock()
	defer t.mu.Unlock()

	n := 0

	for _, address := range peers {
		if _, ok := t.clients[address]; ok {
			continue
		}
		go t.spawnPeer(ctx, address)
		n++
	}

	log.Printf("task.spawnPeers: %d peers spawned", n)
}

func (t *Task) spawnPeer(ctx context.Context, address string) {
	conn, err := net.DialTimeout("tcp", address, time.Second*10)
	if err != nil {
		log.Printf("dial peer error: %v\n", err)
		return
	}

	closeConn := conn
	defer func() {
		if closeConn != nil {
			closeConn.Close()
		}
	}()

	handshake, err := peer.HandshakeOutgoing(conn, 10, t.File.InfoHash(), trackercommon.MyPeerID)
	if err != nil {
		log.Printf("handshake failed: %v\n", err)
		return
	}

	c := peer.Peer{
		Ctx:        ctx,
		HerPeerID:  handshake.PeerID,
		PM:         t.PM,
		MyBitField: t.BitField,
		InfoHash:   t.File.InfoHash(),
		PeerAddr:   address,
	}

	c.SetConn(conn)

	if err := c.RecvBitField(); err != nil {
		log.Printf("error recv bitbield: %v\n", err)
		return
	}
	if err := c.Send(message.MsgUnChoke, message.UnChoke{}); err != nil {
		log.Printf("error send unchoked: %v\n", err)
		return
	}
	if err := c.Send(message.MsgInterested, message.Interested{}); err != nil {
		log.Printf("error send interested: %v\n", err)
		return
	}

	closeConn = nil

	t.AddClient(&c)
}

func (t *Task) announceOne(ctx context.Context, address string) (int, []string, error) {
	u, err := url.Parse(address)
	if err != nil {
		log.Printf("task.announce: malformed address: %v", err)
		return 0, nil, err
	}

	switch u.Scheme {
	case `http`, `https`:
		tr := trackertcpclient.Client{
			Address:  address,
			InfoHash: t.InfoHash,
			MyPeerID: trackercommon.MyPeerID,
		}
		resp, err := tr.Announce(ctx)
		if err != nil {
			log.Printf("Announce failed: %v", err)
			return 0, nil, err
		}
		if resp.FailureReason != "" {
			log.Printf("Announce: failure reason: %s", resp.FailureReason)
			return 0, nil, err
		}
		peers := make([]string, 0, len(resp.Peers))
		for _, peer := range resp.Peers {
			peers = append(peers, fmt.Sprintf(`%s:%d`, peer.IP, peer.Port))
		}
		return resp.Interval, peers, nil
	case `udp`:
		tr := trackerudpclient.Client{
			Address:  address,
			InfoHash: t.InfoHash,
			MyPeerID: trackercommon.MyPeerID,
		}
		resp, err := tr.Announce(ctx)
		if err != nil {
			log.Printf("Announce failed: %v", err)
			return 0, nil, err
		}
		return int(resp.Interval), resp.Peers, nil
	default:
		log.Printf("task.announce: unknown address: %s\n", address)
		return 0, nil, fmt.Errorf("task.announce: unknown address")
	}
}
