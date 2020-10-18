package trackertcpcommon

import (
	"fmt"

	"github.com/movsb/torrent/pkg/common"
)

// Peer ...
type Peer struct {
	ID   common.PeerID `bencode:"peer id"`
	IP   string        `bencode:"ip"`
	Port int           `bencode:"port"`
}

func (p Peer) String() string {
	return fmt.Sprintf(`%v (%s:%d)`, p.ID.String(), p.IP, p.Port)
}
