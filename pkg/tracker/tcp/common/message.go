package trackertcpcommon

// AnnounceResponse ...
type AnnounceResponse struct {
	FailureReason string `bencode:"failure reason"`
	Interval      int    `bencode:"interval,omitempty"`
	Peers         []Peer `bencode:"peers,omitempty"`
}
