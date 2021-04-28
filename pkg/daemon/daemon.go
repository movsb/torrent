package main

import (
	"time"

	"github.com/movsb/torrent/pkg/daemon/task"
	"github.com/movsb/torrent/pkg/seeder"
	trackercommon "github.com/movsb/torrent/pkg/tracker/common"
)

// Daemon ...
type Daemon struct {
}

func main() {
	tm := task.NewManager()

	seeder := seeder.Server{
		Address:     `localhost:8888`,
		MyPeerID:    trackercommon.MyPeerID,
		LoadTorrent: tm,
	}

	tm.CreateTask("ubuntu.torrent", ".", 0xFF)
	//tm.CreateTask("ubuntu.torrent", ".", 0x00)

	if err := seeder.Run(); err != nil {
		panic(err)
	}
	time.Sleep(time.Hour)
}
