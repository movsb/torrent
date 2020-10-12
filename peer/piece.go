package peer

// SinglePieceData ...
type SinglePieceData struct {
	Index  int
	Hash   []byte
	Length int
	Data   []byte

	downloaded int
	requested  int
}
