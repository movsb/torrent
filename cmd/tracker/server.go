package tracker

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	tcptrackerserver "github.com/movsb/torrent/pkg/tracker/server/tcp"
	"github.com/spf13/cobra"
)

func runServer(cmd *cobra.Command, args []string) error {
	s := tcptrackerserver.NewTCPTrackerServer(args[0])

	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Run(ctx); err != nil {
		cancel()
		return err
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT)
	signal.Notify(quit, syscall.SIGKILL)
	signal.Notify(quit, syscall.SIGTERM)
	<-quit
	close(quit)

	cancel()

	return nil
}
