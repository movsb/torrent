package file

import (
	"log"
	"os"

	"github.com/movsb/torrent/file"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// AddCommands ...
func AddCommands(root *cobra.Command) {
	fileCmd := &cobra.Command{
		Use:   `file`,
		Short: `Torrent file related commands`,
	}
	root.AddCommand(fileCmd)

	infoCmd := &cobra.Command{
		Use:   `info <torrent-file>`,
		Short: `Show info about a torrent file`,
		Args:  cobra.ExactArgs(1),
		RunE:  fileInfo,
	}
	fileCmd.AddCommand(infoCmd)

	listFilesCmd := &cobra.Command{
		Use:   `list <torrent-file>`,
		Short: `List files in torrent file.`,
		Args:  cobra.ExactArgs(1),
		RunE:  fileList,
	}
	fileCmd.AddCommand(listFilesCmd)
}

func fileInfo(cmd *cobra.Command, args []string) error {
	tf, err := file.ParseFile(args[0])
	if err != nil {
		log.Println(err)
		return err
	}
	yaml.NewEncoder(os.Stdout).Encode(map[string]interface{}{
		`Name`:        tf.Name,
		`Announce`:    tf.Announce,
		`Length`:      tf.Length,
		`FileCount`:   len(tf.Files),
		`PieceLength`: tf.PieceLength,
		`PieceCount`:  tf.PieceHashes.Len(),
	})
	return nil
}

func fileList(cmd *cobra.Command, args []string) error {
	tf, err := file.ParseFile(args[0])
	if err != nil {
		log.Println(err)
		return err
	}
	yaml.NewEncoder(os.Stdout).Encode(tf.Files)
	return nil
}
