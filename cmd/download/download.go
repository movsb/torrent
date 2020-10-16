package download

import (
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/movsb/torrent/file"
	"github.com/movsb/torrent/message"
	"github.com/movsb/torrent/peer"
	tracker "github.com/movsb/torrent/tracker/tcp"
	udptracker "github.com/movsb/torrent/tracker/udp"
	"github.com/spf13/cobra"
)

// AddCommands ...
func AddCommands(root *cobra.Command) {
	downloadCmd := &cobra.Command{
		Use:   "download <torrent>",
		Short: "Download the specified torrent.",
		Args:  cobra.ExactArgs(1),
		RunE:  downloadTorrent,
	}
	downloadCmd.Flags().StringP("tracker", "t", "", "use this tracker")
	root.AddCommand(downloadCmd)
}

func downloadTorrent(cmd *cobra.Command, args []string) error {
	f, err := file.ParseFile(args[0])
	if err != nil {
		return err
	}

	trackerURL := f.Announce
	if t, _ := cmd.Flags().GetString("tracker"); t != "" {
		trackerURL = t
	}

	u, err := url.Parse(trackerURL)
	if err != nil {
		return err
	}

	var peers []string

	switch u.Scheme {
	case "http", "https":
		t := tracker.TCPTracker{
			Address:  trackerURL,
			InfoHash: f.InfoHash(),
		}
		r, err := t.Announce()
		if err != nil {
			return err
		}
		for _, p := range r.Peers {
			peers = append(peers, fmt.Sprintf("%s:%d", p.IP, p.Port))
		}
	case "udp":
		t := udptracker.UDPTracker{
			Address:  trackerURL,
			InfoHash: f.InfoHash(),
			MyPeerID: tracker.MyPeerID,
		}
		r, err := t.Announce()
		if err != nil {
			return err
		}
		peers = r.Peers
	}

	nPieces := f.PieceHashes.Len()
	chPieces := make(chan peer.SinglePieceData, nPieces)
	for i := 0; i < nPieces-1; i++ {
		chPieces <- peer.SinglePieceData{
			Index:  i,
			Hash:   f.PieceHashes.Index(i),
			Length: f.PieceLength,
		}
	}

	lastPieceIndex, lastPieceLength := nPieces-1, f.PieceLength
	if remain := int(f.Length % int64(f.PieceLength)); remain != 0 {
		lastPieceLength = remain
	}
	chPieces <- peer.SinglePieceData{
		Index:  lastPieceIndex,
		Hash:   f.PieceHashes.Index(lastPieceIndex),
		Length: lastPieceLength,
	}

	chResult := make(chan peer.SinglePieceData)

	wg := &sync.WaitGroup{}

	nClient := int32(len(peers))

	for _, p := range peers {
		wg.Add(1)
		go func(p string) {
			defer func() {
				wg.Done()
				atomic.AddInt32(&nClient, -1)
			}()

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

			if err := c.Send(message.MsgUnChoke, message.UnChoke{}); err != nil {
				fmt.Println(err)
				return
			}

			if err := c.Send(message.MsgInterested, message.Interested{}); err != nil {
				fmt.Println(err)
				return
			}

			if err := c.Download(chPieces, chResult); err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println(`client exit`)
		}(p)
	}

	ifm := peer.NewIndexFileManager(f.Name, f.Single, f.Files, f.PieceLength, f.PieceHashes)

	donePieces := 0
	for donePieces < nPieces {
		piece := <-chResult
		err := ifm.WritePiece(piece.Index, piece.Data)
		if err != nil {
			panic(fmt.Errorf("WritePiece failed: %s", err))
		}
		donePieces++
		percent := float64(donePieces) / float64(f.PieceHashes.Len()) * 100
		fmt.Printf("%0.2f piece downloaded: %d / %d | %d / %d from %d peers\n",
			percent, donePieces, nPieces,
			donePieces*f.PieceLength, f.Length,
			atomic.LoadInt32(&nClient))
	}

	close(chPieces)
	close(chResult)

	ifm.Close()

	wg.Wait()

	return nil
}
