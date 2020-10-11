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
	ID   PeerID `bencode:"peer id"`
	IP   string `bencode:"ip"`
	Port int    `bencode:"port"`
}

// PeerID ...
type PeerID [PeerIDLength]byte

const (
	PeerIDLength = 20
)

// UnmarshalBencode ...
func (p *PeerID) UnmarshalBencode(r []byte) error {
	if len(r) != 3+PeerIDLength {
		return fmt.Errorf("peer id length error: %d != %d", len(r), PeerIDLength)
	}
	var id string
	if err := bencode.DecodeBytes(r, &id); err != nil {
		return fmt.Errorf("peer id error: %v", err)
	}
	copy(p[:], []byte(id))
	return nil
}

var MyPeerID = makePeerID()

func makePeerID() PeerID {
	var id PeerID
	copy(id[:], []byte(`bt123456781234567890`))
	return id
}

// Announce ...
func (t *Tracker) Announce(infoHash [20]byte, length int64) AnnounceResponse {
	u, err := url.Parse(t.URL)
	if err != nil {
		panic(err)
	}
	a := url.Values{}
	a.Set(`info_hash`, string(infoHash[:]))
	a.Set(`peer_id`, string(MyPeerID[:]))
	a.Set(`port`, `9999`)
	a.Set(`uploaded`, `0`)
	a.Set(`downloaded`, `0`)
	a.Set(`left`, fmt.Sprint(length))
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
