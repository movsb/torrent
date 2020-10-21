package trackertcpclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/movsb/torrent/pkg/common"
	trackercommon "github.com/movsb/torrent/pkg/tracker/common"
	trackertcpcommon "github.com/movsb/torrent/pkg/tracker/tcp/common"
	"github.com/zeebo/bencode"
)

// Client ...
type Client struct {
	Address  string
	InfoHash common.Hash
	MyPeerID common.PeerID
}

// Announce ...
func (t *Client) Announce(ctx context.Context) (*trackertcpcommon.AnnounceResponse, error) {
	u, err := url.Parse(t.Address)
	if err != nil {
		log.Printf("Announce: failed to parse address: %v", err)
		return nil, err
	}
	a := u.Query()
	a.Set(`info_hash`, string(t.InfoHash[:]))
	a.Set(`peer_id`, string(trackercommon.MyPeerID[:]))
	a.Set(`port`, `8888`)
	a.Set(`uploaded`, `0`)
	a.Set(`downloaded`, `0`)
	a.Set(`left`, `0`)
	u.RawQuery = a.Encode()

	log.Printf("Announce: %s\n", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Announce: error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Announce: non-200 status code: %v\n", resp.StatusCode)
		return nil, fmt.Errorf(`trackers returns non-200 status code`)
	}

	var d trackertcpcommon.AnnounceResponse
	if err := bencode.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}
