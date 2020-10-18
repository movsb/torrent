package tracker

import (
	"fmt"
	"net/url"
	"os"

	"github.com/movsb/torrent/file"
	trackercommon "github.com/movsb/torrent/pkg/tracker/common"
	trackertcpclient "github.com/movsb/torrent/pkg/tracker/tcp/client"
	trackerudpclient "github.com/movsb/torrent/pkg/tracker/udp/client"
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
		t := trackertcpclient.Client{
			Address:  tracker,
			InfoHash: f.InfoHash(),
			MyPeerID: trackercommon.MyPeerID,
		}
		r, err := t.Announce()
		if err != nil {
			return err
		}
		yaml.NewEncoder(os.Stdout).Encode(r)
	} else if u.Scheme == "udp" {
		t := trackerudpclient.Client{
			Address:  tracker,
			InfoHash: f.InfoHash(),
			MyPeerID: trackercommon.MyPeerID,
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
