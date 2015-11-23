package agent

import (
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"os"
	p2p "p2p_lib"
	"p2p_lib/bittorrent"
	"p2p_lib/util"
	"path/filepath"
)

type Agent struct {
	DataDir      string
	ListenAddr   string
	DockerClient *docker.Client
	BTClient     bittorrent.BitTorrent
}

type DownloadTask struct {
	ImageName    string
	ImageID      string
	TmpDir       string
	ImageTarPath string
	Torrents     map[string]string
	Mode         string
}

func CreateAgent(dataDir, listenAddr, dockerEndpoint, btClient string) (agent *Agent, err error) {
	if err = initWorkspace(dataDir); err != nil {
		return
	}

	dockerClient, err := docker.NewClient(dockerEndpoint)
	if err != nil {
		return
	}

	err = dockerClient.Ping()
	if err != nil {
		return
	}

	var bt bittorrent.BitTorrent

	if btClient == "anacrolix" {
		bt, err = bittorrent.NewAnacrolix(filepath.Join(dataDir, "package"))
		if err != nil {
			return
		}
	} else if btClient == "ctorrent" {
		//TODO check if ctorrent is installed
		bt, err = bittorrent.NewCtorrent()
		if err != nil {
			return
		}
	} else {
		err = fmt.Errorf("unsupported bt client: %s", btClient)
		return
	}

	agent = &Agent{
		DataDir:      dataDir,
		ListenAddr:   listenAddr,
		DockerClient: dockerClient,
		BTClient:     bt,
	}

	return
}

func initWorkspace(path string) (err error) {
	var paths []string
	paths = append(paths, filepath.Join(path, "image"))
	paths = append(paths, filepath.Join(path, "package", "image"))
	paths = append(paths, filepath.Join(path, "package", "layer"))
	paths = append(paths, filepath.Join(path, "torrent", "image"))
	paths = append(paths, filepath.Join(path, "torrent", "layer"))

	for _, p := range paths {
		if err = os.MkdirAll(p, 644); err != nil {
			return
		}
	}

	return
}

func (a *Agent) ImageTarExist(id string) (exist bool, path string, err error) {
	path = filepath.Join(a.DataDir, "image", id+".tar")
	exist, err = util.FileExist(path)
	return
}

func (a *Agent) TorrentExist(id string, mode string) (exist bool, path string, err error) {
	if mode == p2p.ImageMode {
		path = filepath.Join(a.DataDir, "torrent", "image", id+".torrent")
	} else {
		path = filepath.Join(a.DataDir, "torrent", "layer", id+".torrent")
	}
	exist, err = util.FileExist(path)
	return
}
