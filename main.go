package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/movsb/torrent/cmd/download"
	cmdSeeder "github.com/movsb/torrent/cmd/seeder"
	"github.com/movsb/torrent/cmd/tools"
	"github.com/movsb/torrent/cmd/torrent"
	"github.com/movsb/torrent/cmd/tracker"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   filepath.Base(os.Args[0]),
		Short: `A BitTorrent client.`,
	}

	torrent.AddCommands(rootCmd)
	download.AddCommands(rootCmd)
	tracker.AddCommands(rootCmd)
	cmdSeeder.AddCommands(rootCmd)
	tools.AddCommands(rootCmd)

	if os.Getenv("DEBUG") != "" {
		//rootCmd.SetArgs([]string{"download", "--tracker=localhost:9999/announce", "8ce301d28fe97eed1a6ef7feaf296411b375222f.torrent"})
		rootCmd.SetArgs([]string{"server"})
	}

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
	}
}
