package trackertcpclient

import (
	"fmt"
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
	InfoHash common.InfoHash
	MyPeerID common.PeerID
}

// Announce ...
func (t *Client) Announce() (*trackertcpcommon.AnnounceResponse, error) {
	u, err := url.Parse(t.Address)
	if err != nil {
		return nil, err
	}
	a := url.Values{}
	a.Set(`info_hash`, string(t.InfoHash[:]))
	a.Set(`peer_id`, string(trackercommon.MyPeerID[:]))
	a.Set(`port`, `8888`)
	a.Set(`uploaded`, `0`)
	a.Set(`downloaded`, `0`)
	a.Set(`left`, `0`)
	u.RawQuery = a.Encode()

	fmt.Println(u.String())
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var d trackertcpcommon.AnnounceResponse
	if err := bencode.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}
