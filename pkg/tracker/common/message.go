package trackercommon

import tracker "github.com/movsb/torrent/tracker/tcp"

// AnnounceResponse ...
type AnnounceResponse struct {
	FailureReason string         `bencode:"failure reason,omitempty"`
	Interval      int            `bencode:"interval"`
	Peers         []tracker.Peer `bencode:"peers"`
}
