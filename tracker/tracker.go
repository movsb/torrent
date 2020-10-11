package tracker

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/zeebo/bencode"
)

// Tracker ...
type Tracker struct {
	URL string
}

// AnnounceResponse ...
type AnnounceResponse struct {
	FailureReason string `bencode:"failure reason"`
	Interval      int    `bencode:"interval"`
	Peers         []Peer `bencode:"peers"`
}

// Peer ...
type Peer struct {
	ID   string `bencode:"peer id"`
	IP   string `bencode:"ip"`
	Port int    `bencode:"port"`
}

func (t *Tracker) Announce(infoHash [20]byte) AnnounceResponse {
	u, err := url.Parse(t.URL)
	if err != nil {
		panic(err)
	}
	a := url.Values{}
	a.Set(`info_hash`, string(infoHash[:]))
	a.Set(`peer_id`, `bt123456781234567890`)
	a.Set(`port`, `9999`)
	a.Set(`uploaded`, `0`)
	a.Set(`downloaded`, `0`)
	a.Set(`left`, `1`)
	u.RawQuery = a.Encode()

	fmt.Println(u.String())
	resp, err := http.Get(u.String())
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var d AnnounceResponse
	if err := bencode.NewDecoder(resp.Body).Decode(&d); err != nil {
		panic(err)
	}
	return d
}
