package tracker

import (
	"fmt"
	"net/url"
	"os"

	"github.com/movsb/torrent/file"
	tcptracker "github.com/movsb/torrent/tracker/tcp"
	udptracker "github.com/movsb/torrent/tracker/udp"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// AddCommands ...
func AddCommands(root *cobra.Command) {
	trackerCmd := &cobra.Command{
		Use: "tracker",
	}
	root.AddCommand(trackerCmd)

	testCmd := &cobra.Command{
		Use:  "test <tracker> <torrent>",
		Args: cobra.ExactArgs(2),
		RunE: testTracker,
	}
	trackerCmd.AddCommand(testCmd)

	runServerCmd := &cobra.Command{
		Use:  `server <endpoint>`,
		Args: cobra.ExactArgs(1),
		RunE: runServer,
	}
	trackerCmd.AddCommand(runServerCmd)
}

func testTracker(cmd *cobra.Command, args []string) error {
	tracker := args[0]
	u, err := url.Parse(tracker)
	if err != nil {
		return err
	}

	torrent := args[1]
	f, err := file.ParseFile(torrent)
	if err != nil {
		return err
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		t := tcptracker.TCPTracker{
			Address:  tracker,
			InfoHash: f.InfoHash(),
			MyPeerID: tcptracker.MyPeerID,
		}
		r, err := t.Announce()
		if err != nil {
			return err
		}
		yaml.NewEncoder(os.Stdout).Encode(r)
	} else if u.Scheme == "udp" {
		t := udptracker.UDPTracker{
			Address:  tracker,
			InfoHash: f.InfoHash(),
			MyPeerID: tcptracker.MyPeerID,
		}
		r, err := t.Announce()
		if err != nil {
			return err
		}
		yaml.NewEncoder(os.Stdout).Encode(r)
	} else {
		return fmt.Errorf("invalid tracker protocol")
	}

	return nil
}
