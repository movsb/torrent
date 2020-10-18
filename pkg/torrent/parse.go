package torrent

import (
	"os"

	"github.com/zeebo/bencode"
)

func ParseFile(path string) (*File, error) {
	fp, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	f := _File{}
	if err := bencode.NewDecoder(fp).Decode(&f); err != nil {
		panic(err)
	}

	return f.convert()
}

func ParseFileToInterface(path string) (interface{}, error) {
	fp, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	var f interface{}
	if err := bencode.NewDecoder(fp).Decode(&f); err != nil {
		panic(err)
	}
	return f, nil
}
