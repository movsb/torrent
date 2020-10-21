package main

import (
	"time"

	"github.com/movsb/torrent/pkg/daemon/task"
)

// Daemon ...
type Daemon struct {
}

func main() {
	tm := task.NewManager()

	//seeder := seeder.Server{
	//	Address:     `localhost:8888`,
	//	MyPeerID:    trackercommon.MyPeerID,
	//	LoadTorrent: tm,
	//}

	//tm.CreateTask("8ce301d28fe97eed1a6ef7feaf296411b375222f.torrent", ".", 0xFF)
	tm.CreateTask("ubuntu.torrent", ".", 0x00)

	// if err := seeder.Run(); err != nil {
	// 	panic(err)
	// }
	time.Sleep(time.Hour)
}
