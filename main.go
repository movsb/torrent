package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/movsb/torrent/file"
	"github.com/movsb/torrent/peer"
	"github.com/movsb/torrent/tracker"
)

func main() {
	n := `ubuntu.torrent`
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
	r := t.Announce(f.InfoHash(), f.Length)
	if len(r.Peers) == 0 {
		return
	}

	wg := &sync.WaitGroup{}

	for _, p := range r.Peers {
		wg.Add(1)
		go func(p tracker.Peer) {
			defer wg.Done()

			c := peer.Client{
				Peer:     p,
				InfoHash: f.InfoHash(),
			}

			defer c.Close()

			if err := c.Handshake(); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(`handshake success`)

			if err := c.RecvBitField(); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(`received bitfield`)

		}(p)
		break
	}

	wg.Wait()
}
