package tracker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

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

func (p Peer) String() string {
	return fmt.Sprintf(`%v (%s:%d)`, p.ID.String(), p.IP, p.Port)
}

// PeerID ...
type PeerID [PeerIDLength]byte

func (p PeerID) String() string {
	b, _ := json.Marshal(p)
	s := string(b[1 : len(b)-1])
	s = strings.ReplaceAll(s, `\"`, `"`)
	return s
}

// PeerIDLength ...
const PeerIDLength = 20

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

// MyPeerID ...
var MyPeerID = makePeerID()

func makePeerID() PeerID {
	var id PeerID
	copy(id[:], []byte(`dev-bt12345678123456`))
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
