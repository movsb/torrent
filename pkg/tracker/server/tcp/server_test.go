package tcptrackerserver

import (
	"context"
	"testing"
)

func TestTCPTrackerServer(t *testing.T) {
	s := TCPTrackerServer{
		endpoint: `localhost:9999/announce`,
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}

	cancel()
}
