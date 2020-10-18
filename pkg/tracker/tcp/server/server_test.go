package trackertcpserver

import (
	"context"
	"testing"
)

func TestServer(t *testing.T) {
	s := Server{
		endpoint: `localhost:9999/announce`,
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}

	cancel()
}
