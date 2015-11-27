package bittorrent

import (
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"os"
	"runtime"
	"time"
)

type Anacrolix struct {
	Client *torrent.Client
}

func NewAnacrolix(dataDir string) (anacrolix *Anacrolix, err error) {
	cfg := &torrent.Config{
		DataDir: dataDir,
		Seed:    true,
		NoDHT:   true,
	}

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return
	}

	anacrolix = &Anacrolix{
		Client: client,
	}
	return
}

func (a *Anacrolix) CreateTorrent(path, torrentPath string, trackers []string) (err error) {
	builder := metainfo.Builder{}
	builder.AddFile(path)
	builder.AddAnnounceGroup(trackers)

	batch, err := builder.Submit()
	if err != nil {
		return
	}

	torrentFile, err := os.Create(torrentPath)
	if err != nil {
		return
	}
	defer torrentFile.Close()

	errs, _ := batch.Start(torrentFile, runtime.NumCPU())
	err = <-errs
	if err != nil {
		return
	}

	return
}

func (a *Anacrolix) Download(path, torrentPath string, configs map[string]string) (err error) {
	t, err := a.Client.AddTorrentFromFile(torrentPath)
	if err != nil {
		return
	}
	<-t.GotInfo()
	t.DownloadAll()

	for {
		if t.Length() == t.BytesCompleted() {
			break
		}
		time.Sleep(time.Second * 1)
	}

	return
}
