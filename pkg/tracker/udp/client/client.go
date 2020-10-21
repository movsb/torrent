package trackerudpclient

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"time"

	"github.com/movsb/torrent/pkg/common"
	"github.com/movsb/torrent/pkg/utils"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func makeTransactionID() uint32 {
	return rand.Uint32()
}

// Client ...
type Client struct {
	Address  string
	InfoHash common.Hash
	MyPeerID common.PeerID
	conn     *net.UDPConn
}

// Announce ...
// Ctx isn't used.
func (t *Client) Announce(ctx context.Context) (*AnnounceResponse, error) {
	if err := t.dial(); err != nil {
		return nil, err
	}
	defer t.conn.Close()

	connResp, err := t.connect()
	if err != nil {
		return nil, err
	}
	announceResp, err := t.announce(connResp.ConnectionID)
	if err != nil {
		return nil, err
	}
	return announceResp, nil
}

func (t *Client) dial() error {
	u, err := url.Parse(t.Address)
	if err != nil {
		return err
	}
	if u.Scheme != "udp" {
		return fmt.Errorf("not an udp address")
	}
	dstAddr, err := net.ResolveUDPAddr("udp", u.Host)
	if err != nil {
		return fmt.Errorf("resolve udp address failed: %v", err)
	}
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		return fmt.Errorf("dial udp address failed: %v", err)
	}
	t.conn = conn
	return nil
}

// Connect ...
func (t *Client) connect() (*ConnectResponse, error) {
	defer t.conn.SetDeadline(time.Time{})

	req := ConnectRequest{
		ProtocolID:    protocolID,
		Action:        ActionConnect,
		TransactionID: makeTransactionID(),
	}
	b, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal error: %v", err)
	}

	// How to use context here?
	utils.SetDeadlineSeconds(t.conn, 10)
	if _, err = t.conn.Write(b); err != nil {
		return nil, fmt.Errorf("connect error: %v", err)
	}

	b = make([]byte, 16)
	utils.SetDeadlineSeconds(t.conn, 10)
	_, err = t.conn.Read(b)
	if err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}

	resp := ConnectResponse{}
	if err = resp.Unmarshal(b); err != nil {
		return nil, fmt.Errorf("ConnectResponse error: %v", err)
	}
	if resp.TransactionID != req.TransactionID {
		return nil, fmt.Errorf("TransactionID mismatch")
	}
	if resp.Action != ActionConnect {
		return nil, fmt.Errorf("Action mismatch")
	}
	return &resp, nil
}

func (t *Client) announce(connectionID uint64) (*AnnounceResponse, error) {
	defer t.conn.SetDeadline(time.Time{})

	req := AnnounceRequest{
		ConnectionID:  connectionID,
		Action:        ActionAnnounce,
		TransactionID: makeTransactionID(),
		InfoHash:      t.InfoHash,
		PeerID:        t.MyPeerID,
		Downloaded:    0,
		Left:          0,
		Uploaded:      0,
		Event:         EventNone,
		IP:            net.IPv4zero,
		Key:           0,
		NumWant:       -1,
		Port:          8888,
	}
	b, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal error: %v", err)
	}

	utils.SetDeadlineSeconds(t.conn, 10)
	if _, err := t.conn.Write(b); err != nil {
		return nil, fmt.Errorf("announce error: %v", err)
	}

	b = make([]byte, 65536)
	utils.SetDeadlineSeconds(t.conn, 10)
	n, err := t.conn.Read(b)
	if err != nil {
		return nil, fmt.Errorf("read announce failed: %v", err)
	}

	resp := AnnounceResponse{}
	if err = resp.Unmarshal(b[:n]); err != nil {
		return nil, fmt.Errorf("AnnounceResponse error: %v", err)
	}

	if resp.TransactionID != req.TransactionID {
		return nil, fmt.Errorf("TransactionID mismatch")
	}
	if resp.Action != ActionAnnounce {
		return nil, fmt.Errorf("Action mismatch")
	}

	return &resp, nil
}
