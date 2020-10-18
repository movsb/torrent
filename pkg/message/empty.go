package message

import "fmt"

// _Empty ...
type _Empty struct{}

var _ Message = &_Empty{}

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
