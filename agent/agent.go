package agent

import (
	"errors"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"os"
	"path/filepath"
	"registry_p2p/bittorrent"
	"registry_p2p/utils"
	"sync"
)

type Agent struct {
	sync.Mutex

	DataDir      string
	Port         string
	DockerClient *docker.Client
	BTClient     bittorrent.BitTorrent

	TaskPool map[string]chan struct{}
}

func NewAgent(dataDir, port, dockerEndpoint, btClient string) (agent *Agent, err error) {
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

	if btClient == "builtin" {
		bt, err = bittorrent.NewBuiltin(filepath.Join(dataDir, "package"))
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
		Port:         port,
		DockerClient: dockerClient,
		BTClient:     bt,
		TaskPool:     make(map[string]chan struct{}),
	}
	return
}

func initWorkspace(path string) (err error) {
	var paths []string
	paths = append(paths, filepath.Join(path, "package", "image"))
	paths = append(paths, filepath.Join(path, "package", "layer"))
	paths = append(paths, filepath.Join(path, "package", "multi_layer"))
	paths = append(paths, filepath.Join(path, "torrent", "image"))
	paths = append(paths, filepath.Join(path, "torrent", "layer"))
	paths = append(paths, filepath.Join(path, "torrent", "multi_layer"))

	for _, p := range paths {
		if err = os.MkdirAll(p, 644); err != nil {
			return
		}
	}

	return
}

func (a *Agent) TorrentExist(id string, typee string) (exist bool, path string, err error) {
	switch typee {
	case "image":
		path = filepath.Join(a.DataDir, "torrent", "image", id+".torrent")
	case "layer":
		path = filepath.Join(a.DataDir, "torrent", "layer", id+".torrent")
	case "multi_layer":
		path = filepath.Join(a.DataDir, "torrent", "multi_layer", id+".torrent")
	}
	exist, err = utils.FileExist(path)
	return
}

func (a *Agent) PackageExist(id string, typee string) (exist bool, path string, err error) {
	switch typee {
	case "image":
		path = filepath.Join(a.DataDir, "package", "image", id+".tar.gz")
	case "layer":
		path = filepath.Join(a.DataDir, "package", "layer", id+".tar.gz")
	case "multi_layer":
		path = filepath.Join(a.DataDir, "package", "multi_layer", id+".tar.gz")
	}
	exist, err = utils.FileExist(path)

	return
}

func (a *Agent) PoolAdd(key string) (c chan struct{}, err error) {
	a.Lock()
	defer a.Unlock()

	if c, exists := a.TaskPool[key]; exists {
		return c, errors.New("task is already in progress")
	}

	c = make(chan struct{})
	a.TaskPool[key] = c
	return
}

func (a *Agent) PoolDelete(key string) {
	a.Lock()
	defer a.Unlock()

	if c, exists := a.TaskPool[key]; exists {
		close(c)
		delete(a.TaskPool, key)
	}
	return
}
