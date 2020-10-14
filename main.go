package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/movsb/torrent/cmd/download"
	cmdfile "github.com/movsb/torrent/cmd/file"
	"github.com/movsb/torrent/cmd/tracker"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   filepath.Base(os.Args[0]),
		Short: `A BitTorrent client.`,
	}

	cmdfile.AddCommands(rootCmd)
	download.AddCommands(rootCmd)
	tracker.AddCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
	}
}
