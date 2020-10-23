package download

import (
	"time"

	"github.com/movsb/torrent/pkg/daemon/task"
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
	tm := task.NewManager()
	tm.CreateTask(args[0], ".", 0x00)
	time.Sleep(time.Hour)
	return nil
}
