package trackerudpclient

import (
	"os"
	"testing"

	"github.com/movsb/torrent/pkg/torrent"
	trackercommon "github.com/movsb/torrent/pkg/tracker/common"
	"gopkg.in/yaml.v3"
)

func TestClient(t *testing.T) {
	f, err := torrent.ParseFile("../../ubuntu.torrent")
	if err != nil {
		t.Fatal(err)
	}

	ut := &Client{
		Address:  `udp://tracker.leechers-paradise.org:6969`,
		InfoHash: f.InfoHash(),
		MyPeerID: trackercommon.MyPeerID,
	}

	resp, err := ut.Announce()
	if err != nil {
		t.Fatal(err)
	}

	yaml.NewEncoder(os.Stdout).Encode(resp)
}
