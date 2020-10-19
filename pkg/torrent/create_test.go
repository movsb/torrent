package torrent

import (
	"os"
	"testing"
)

func TestWalk(t *testing.T) {
	c := NewCreator(".")
	if err := c.Create(os.Stdout); err != nil {
		t.Fatal(err)
	}
}
