package tools

import (
	"github.com/movsb/torrent/cmd/tools/bencode"
	"github.com/spf13/cobra"
)

// AddCommands ...
func AddCommands(parent *cobra.Command) {
	toolsCmd := &cobra.Command{
		Use:   `tools`,
		Short: `Some tools`,
	}
	parent.AddCommand(toolsCmd)

	bencode.AddCommands(toolsCmd)
}
