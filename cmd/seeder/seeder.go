package cmdseeder

import (
	"github.com/movsb/torrent/pkg/seeder"
	"github.com/spf13/cobra"
)

func AddCommands(root *cobra.Command) {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Runs seeder server",
		RunE:  runSeeder,
	}
	root.AddCommand(serverCmd)
}

func runSeeder(cmd *cobra.Command, args []string) error {
	s := seeder.Server{
		Address: `localhost:8888`,
	}
	return s.Run()
}
