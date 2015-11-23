package bittorrent

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

type Ctorrent struct {
}

func NewCtorrent() (ctorrent *Ctorrent, err error) {
	//TODO check ctorrent is already installed
	ctorrent = &Ctorrent{}
	return
}

func (c *Ctorrent) CreateTorrent(path, torrentPath string, trackers []string) (err error) {
	ctorrentCmd := fmt.Sprintf("ctorrent -t -u %s -s %s %s", trackers[0], torrentPath, path)

	cmd := exec.Command("bash", "-c", ctorrentCmd)
	if err = cmd.Run(); err != nil {
		return
	}
	return
}

func (c *Ctorrent) Download(torrents, configs map[string]string) (err error) {
	if configs["target"] == "manager" {
		err = downloadForManager(torrents)
	} else {
		err = downloadForAgent(torrents)
	}

	return
}

func downloadForManager(torrents map[string]string) (err error) {
	for path, torrentPath := range torrents {
		if err = btDownload(path, torrentPath, true, 1); err != nil {
			return
		}
	}

	return
}

func downloadForAgent(torrents map[string]string) (err error) {
	var wg sync.WaitGroup
	wg.Add(len(torrents))

	for path, torrentPath := range torrents {
		go func(path, torrentPath string) {
			defer wg.Done()

			log.Printf("starting download %s", path)
			if err = btDownload(path, torrentPath, false, 0); err != nil {
				log.Printf("download %s error: %s", path, err.Error())
				return
			}
			log.Printf("complete download %s", path)

			log.Printf("seeding %s", path)
			if err = btDownload(path, torrentPath, true, 1); err != nil {
				log.Printf("seeding %s error: %s", path, err.Error())
				return
			}

		}(path, torrentPath)
	}

	wg.Wait()

	return
}

func btDownload(path, torrentPath string, daemon bool, seedTime int) (err error) {
	ctorrentCmd := "ctorrent "

	if daemon {
		ctorrentCmd = ctorrentCmd + "-d "
	}

	ctorrentCmd = fmt.Sprintf(ctorrentCmd+"-s %s -e %d %s", path, seedTime, torrentPath)

	cmd := exec.Command("bash", "-c", ctorrentCmd)
	if err = cmd.Run(); err != nil {
		return
	}

	return nil
}
