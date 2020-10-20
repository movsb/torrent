package bencode

import "github.com/spf13/cobra"

// AddCommands ...
func AddCommands(parent *cobra.Command) {
	bencodeCmd := &cobra.Command{
		Use:   `bencode`,
		Short: `Bencode encode and decode`,
	}
	parent.AddCommand(bencodeCmd)

	decodeCmd := &cobra.Command{
		Use:   `decode <file>`,
		Short: `Decode bencode from file.`,
		Args:  cobra.ExactArgs(1),
		Run:   decode,
	}
	bencodeCmd.AddCommand(decodeCmd)
}
