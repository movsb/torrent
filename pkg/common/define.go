package common

import "bytes"

// InfoHash ...
type InfoHash [20]byte

// Equal ...
func (ih InfoHash) Equal(other InfoHash) bool {
	return bytes.Equal(ih[:], other[:])
}

// Set ...
func (ih *InfoHash) Set(other []byte) {
	if len(other) != 20 {
		panic("info_hash length must be 20")
	}
	copy(ih[:], other)
}

// Copy ...
func (ih *InfoHash) Copy(other InfoHash) {
	copy(ih[:], other[:])
}
