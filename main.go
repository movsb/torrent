package main

import (
	"os"

	"github.com/movsb/torrent/file"
	"github.com/movsb/torrent/tracker"
)

func main() {
	n := `debian.torrent`
	if len(os.Args) == 2 {
		n = os.Args[1]
	}
	f, err := file.ParseFile(n)
	if err != nil {
		panic(err)
	}
	t := tracker.Tracker{
		URL: f.Announce,
	}
	t.Announce(f.InfoHash())
}
