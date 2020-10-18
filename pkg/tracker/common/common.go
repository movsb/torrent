package trackercommon

import "github.com/movsb/torrent/pkg/common"

// MyPeerID ...
var MyPeerID = makePeerID()

func makePeerID() common.PeerID {
	var id common.PeerID
	copy(id[:], []byte(`dev-bt12345678123457`))
	return id
}
