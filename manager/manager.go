package manager

import (
	"errors"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"net"
	"os"
	"path/filepath"
	"registry_p2p/bittorrent"
	sch "registry_p2p/manager/scheduler"
	"registry_p2p/utils"
	"sync"
)

type Manager struct {
	sync.Mutex

	DataDir      string
	Port         string
	DockerClient *docker.Client
	Trackers     []string

	BTClient  bittorrent.BitTorrent
	Scheduler sch.Scheduler

	FileServerPrefix string

	TaskPool map[string]chan struct{}
}

func NewManager(dataDir, port, dockerEndpoint, btClient, scheduler string, trackers []string) (manager *Manager, err error) {
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

	var s sch.Scheduler

	if scheduler == "batch" {
		s = sch.NewBatch()
	} else {
		err = fmt.Errorf("unsupported scheduler: %s", scheduler)
		return
	}

	if len(trackers) == 0 {
		err = errors.New("tracker is nil")
		return
	}

	ip, err := getIP()
	if err != nil {
		return
	}

	manager = &Manager{
		DataDir:          dataDir,
		Port:             port,
		DockerClient:     dockerClient,
		BTClient:         bt,
		Scheduler:        s,
		Trackers:         trackers,
		FileServerPrefix: "http://" + ip + ":" + port + "/torrent/",
		TaskPool:         make(map[string]chan struct{}),
	}

	return
}

func getIP() (ip string, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				return
			}
		}
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

func (m *Manager) PackageExist(id string, typee string) (exist bool, path string, err error) {
	switch typee {
	case "image":
		path = filepath.Join(m.DataDir, "package", "image", id+".tar.gz")
	case "layer":
		path = filepath.Join(m.DataDir, "package", "layer", id+".tar.gz")
	case "multi_layer":
		path = filepath.Join(m.DataDir, "package", "multi_layer", id+".tar.gz")
	default:
		err = errors.New("unsupport type")
	}
	exist, err = utils.FileExist(path)

	return
}

func (m *Manager) TorrentExist(id string, typee string) (exist bool, path string, err error) {
	switch typee {
	case "image":
		path = filepath.Join(m.DataDir, "torrent", "image", id+".torrent")
	case "layer":
		path = filepath.Join(m.DataDir, "torrent", "layer", id+".torrent")
	case "multi_layer":
		path = filepath.Join(m.DataDir, "torrent", "multi_layer", id+".torrent")
	default:
		err = errors.New("unsupport type")
	}
	exist, err = utils.FileExist(path)
	return
}

func (m *Manager) PoolAdd(key string) (c chan struct{}, err error) {
	m.Lock()
	defer m.Unlock()

	if c, exists := m.TaskPool[key]; exists {
		return c, errors.New("task is already in progress")
	}

	c = make(chan struct{})
	m.TaskPool[key] = c
	return
}

func (m *Manager) PoolDelete(key string) {
	m.Lock()
	defer m.Unlock()

	if c, exists := m.TaskPool[key]; exists {
		close(c)
		delete(m.TaskPool, key)
	}
	return
}
