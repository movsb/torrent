package main

import (
	"github.com/movsb/torrent/file"
	"github.com/movsb/torrent/tracker"
)

func main() {
	f, _ := file.ParseFile(`ubuntu.torrent`)
	t := tracker.Tracker{
		URL: f.Announce,
	}
	t.Announce(f.InfoHash())
}
