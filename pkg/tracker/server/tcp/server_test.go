package tcptrackerserver

import (
	"context"
	"testing"
)

func TestTCPTrackerServer(t *testing.T) {
	s := TCPTrackerServer{
		Address: `localhost:9999`,
		Path:    `/announce`,
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}

	cancel()
}
