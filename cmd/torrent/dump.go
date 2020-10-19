package torrent

import (
	"fmt"
	"log"
	"os"

	"github.com/movsb/torrent/pkg/torrent"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func dumpFile(cmd *cobra.Command, args []string) error {
	fi, err := torrent.ParseFileToInterface(args[0])
	if err != nil {
		log.Println(err)
		return err
	}

	if hasPieceHashes, _ := cmd.Flags().GetBool("piece-hashes"); !hasPieceHashes {
		if fii, ok := fi.(map[string]interface{}); ok {
			if infoi, ok := fii["info"].(map[string]interface{}); ok {
				delete(infoi, "pieces")
			}
		}
	}

	yaml.NewEncoder(os.Stdout).Encode(fi)
	return nil
}

func infoHashCmd(cmd *cobra.Command, args []string) error {
	for _, path := range args {
		tf, err := torrent.ParseFile(path)
		if err != nil {
			log.Println(err)
			return err
		}
		fmt.Println(tf.InfoHash(), path)
	}
	return nil
}
