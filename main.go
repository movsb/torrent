package main

import (
	"fmt"

	"github.com/movsb/torrent/file"
)

func main() {
	f, _ := file.ParseFile(`debian.torrent`)
	fmt.Println(f.InfoHash())
	fmt.Println(f)
}
