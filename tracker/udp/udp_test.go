package tracker

import (
	"os"
	"testing"

	"github.com/movsb/torrent/file"
	"github.com/movsb/torrent/tracker"
	"gopkg.in/yaml.v3"
)

func TestUDPTracker(t *testing.T) {
	f, err := file.ParseFile("../../ubuntu.torrent")
	if err != nil {
		t.Fatal(err)
	}

	ut := &UDPTracker{
		Address:  `udp://tracker.leechers-paradise.org:6969`,
		InfoHash: f.InfoHash(),
		MyPeerID: tracker.MyPeerID,
	}

	resp, err := ut.Announce()
	if err != nil {
		t.Fatal(err)
	}

	yaml.NewEncoder(os.Stdout).Encode(resp)
}
