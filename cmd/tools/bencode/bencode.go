package bencode

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/zeebo/bencode"
	"gopkg.in/yaml.v3"
)

var decodeFlags struct {
}

func decode(cmd *cobra.Command, args []string) {
	var r io.ReadCloser
	path := args[0]
	switch path {
	case "-":
		r = ioutil.NopCloser(os.Stdin)
	default:
		fp, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open: %s: %v\n", path, err)
			os.Exit(1)
		}
		r = fp
	}
	defer r.Close()

	var v interface{}
	if err := bencode.NewDecoder(r).Decode(&v); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding: %v", err)
		os.Exit(1)
	}

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	enc.Encode(v)
}
