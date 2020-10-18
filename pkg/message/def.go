package message

// Message is a BitTorrent message sent between peers.
type Message interface {
	Marshaler
	Unmarshaler
}

// Marshaler ...
type Marshaler interface {
	Marshal() ([]byte, error)
}

// Unmarshaler ...
type Unmarshaler interface {
	Unmarshal([]byte) error
}

// MsgID ...
type MsgID byte

// Known message list
const (
	MsgChoke         = MsgID(0)
	MsgUnChoke       = MsgID(1)
	MsgInterested    = MsgID(2)
	MsgNotInterested = MsgID(3)
	MsgHave          = MsgID(4)
	MsgBitField      = MsgID(5)
	MsgRequest       = MsgID(6)
	MsgPiece         = MsgID(7)
	MsgCancel        = MsgID(8)
)
