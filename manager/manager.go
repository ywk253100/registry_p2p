package manager

import (
	"errors"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"net"
	"net/http"
	"os"
	"p2p_lib"
	"p2p_lib/bittorrent"
	sch "p2p_lib/manager/scheduler"
	"p2p_lib/utils"
	"path/filepath"
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
}

type Task struct {
	ImageID string

	ImageName string   `json:"image"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Email     string   `json:"email"`
	Mode      string   `json:"mode"`
	Hosts     []string `json:"hosts"`

	Owner string

	State      string
	AgentTasks []*AgentTask

	Writer *utils.FlushWriter
}

type AgentTask struct {
	ID    string //ip:port or hostname
	State string
	Items []*AgentTaskItem
}

type AgentTaskItem struct {
	Type string //image, layer or metadata
	URL  string
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
		FileServerPrefix: "http://" + ip + ":" + port + "/",
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
	paths = append(paths, filepath.Join(path, "image"))
	paths = append(paths, filepath.Join(path, "package"))
	paths = append(paths, filepath.Join(path, "torrent"))
	//paths = append(paths, filepath.Join(path, "package", "image"))
	//paths = append(paths, filepath.Join(path, "package", "layer"))
	//paths = append(paths, filepath.Join(path, "torrent", "image"))
	//paths = append(paths, filepath.Join(path, "torrent", "layer"))
	//paths = append(paths, filepath.Join(path, "task", "image"))
	//paths = append(paths, filepath.Join(path, "task", "layer"))
	paths = append(paths, filepath.Join(path, "metadata"))

	for _, p := range paths {
		if err = os.MkdirAll(p, 644); err != nil {
			return
		}
	}

	return
}

func (m *Manager) ImageTarExist(id string) (exist bool, path string, err error) {
	path = filepath.Join(m.DataDir, "image", id+".tar")
	exist, err = util.FileExist(path)
	return
}

func (m *Manager) PackageExist(id string, mode string) (exist bool, path string, err error) {
	if mode == p2p_lib.ImageMode {
		path = filepath.Join(m.DataDir, "package", "image_"+id+".tar.gz")
	} else {
		path = filepath.Join(m.DataDir, "package", "layer_"+id+".tar.gz")
	}
	exist, err = util.FileExist(path)
	return
}

func (m *Manager) TorrentExist(id string, mode string) (exist bool, path string, err error) {
	if mode == p2p_lib.ImageMode {
		path = filepath.Join(m.DataDir, "torrent", "image_"+id+".torrent")
	} else {
		path = filepath.Join(m.DataDir, "torrent", "layer_"+id+".torrent")
	}
	exist, err = util.FileExist(path)
	return
}

func (m *Manager) TaskExist(id string, mode string) (exist bool, path string, err error) {
	if mode == p2p_lib.ImageMode {
		path = filepath.Join(m.DataDir, "task", "image", id+".torrent")
	} else {
		path = filepath.Join(m.DataDir, "task", "layer", id+".tar.gz")
	}
	exist, err = util.FileExist(path)

	return
}

func (m *Manager) MetadataExist(id string) (exist bool, path string, err error) {
	path = filepath.Join(m.DataDir, "metadata", id)
	exist, err = util.FileExist(path)
	return
}
