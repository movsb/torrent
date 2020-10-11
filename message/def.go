package message

import "io"

// Marshaler ...
type Marshaler interface {
	Marshal() ([]byte, error)
}

// Unmarshaler ...
type Unmarshaler interface {
	Unmarshal(r io.Reader) error
}

// MsgID ...
type MsgID byte

const (
	MsgChoke         = MsgID(0)
	MsgUnChoke       = MsgID(1)
	MsgInterested    = MsgID(2)
	MsgNotInterested = MsgID(3)
	MsgHave          = MsgID(4)
	MsgBitField      = MsgID(5)
	MsgRequest       = MsgID(6)
	MsgPieces        = MsgID(7)
	MsgCancel        = MsgID(8)
)
