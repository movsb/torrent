package tracker

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/zeebo/bencode"
)

// TCPTracker ...
type TCPTracker struct {
	Address  string
	InfoHash [20]byte
	MyPeerID PeerID
}

// MyPeerID ...
var MyPeerID = makePeerID()

func makePeerID() PeerID {
	var id PeerID
	copy(id[:], []byte(`dev-bt12345678123456`))
	return id
}

// Announce ...
func (t *TCPTracker) Announce() (*AnnounceResponse, error) {
	u, err := url.Parse(t.Address)
	if err != nil {
		return nil, err
	}
	a := url.Values{}
	a.Set(`info_hash`, string(t.InfoHash[:]))
	a.Set(`peer_id`, string(MyPeerID[:]))
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

	var d AnnounceResponse
	if err := bencode.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}
