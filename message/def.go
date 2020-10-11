package message

import "fmt"

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

// _Empty ...
type _Empty struct{}

var _ Marshaler = &_Empty{}
var _ Unmarshaler = &_Empty{}

// Marshal ...
func (m _Empty) Marshal() ([]byte, error) {
	return nil, nil
}

func (m *_Empty) Unmarshal(b []byte) error {
	if len(b) > 0 {
		return fmt.Errorf("msg length should be zero")
	}
	return nil
}

// Choke ...
type Choke struct {
	_Empty
}

// UnChoke ...
type UnChoke struct {
	_Empty
}

// Interested ...
type Interested struct {
	_Empty
}

// NotInterested ...
type NotInterested struct {
	_Empty
}
