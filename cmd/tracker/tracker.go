package tracker

import (
	"fmt"

	"github.com/movsb/torrent/file"
	trackerpkg "github.com/movsb/torrent/tracker"
	"github.com/spf13/cobra"
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
}

func testTracker(cmd *cobra.Command, args []string) error {
	tracker := args[0]
	torrent := args[1]

	f, err := file.ParseFile(torrent)
	if err != nil {
		return err
	}
	t := trackerpkg.Tracker{
		URL: tracker,
	}
	r := t.Announce(f.InfoHash(), f.Length)
	if len(r.Peers) == 0 {
		return fmt.Errorf("no peers")
	}
	fmt.Println(len(r.Peers))
	return nil
}
