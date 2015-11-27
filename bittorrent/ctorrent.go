package bittorrent

import (
	"fmt"
	"os/exec"
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

func (c *Ctorrent) Download(path, torrentPath string, configs map[string]string) (err error) {
	if configs["target"] == "manager" {
		err = downloadForManager(path, torrentPath)
	} else {
		err = downloadForAgent(path, torrentPath)
	}

	return
}

func downloadForManager(path, torrentPath string) (err error) {
	if err = btDownload(path, torrentPath, true, 1); err != nil {
		return
	}

	return
}

func downloadForAgent(path, torrentPath string) (err error) {
	if err = btDownload(path, torrentPath, false, 0); err != nil {
		return
	}

	if err = btDownload(path, torrentPath, true, 1); err != nil {
		return
	}

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
