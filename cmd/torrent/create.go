package torrent

import (
	"fmt"
	"os"

	"github.com/movsb/torrent/pkg/torrent"
	"github.com/spf13/cobra"
)

func createTorrent(cmd *cobra.Command, args []string) {
	path := args[0]
	c := torrent.NewCreator(path)
	if err := c.Create(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
