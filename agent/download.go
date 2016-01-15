package agent

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	p2p "registry_p2p"
	"strings"
	"sync"
	"time"
)

//TODO remove
var (
	DownloadEnd time.Time
	Lock        = &sync.Mutex{}
)

type DownloadResult struct {
	Err         chan error
	PackagePath string
}

func Download(ag *Agent, task *Task) (results []*DownloadResult, err error) {

	switch task.Mode {
	case p2p.MODE_IMAGE:
		return downloadForImage(ag, task)
	case p2p.MODE_LAYER:
		return downloadForLayer(ag, task)
	case p2p.MODE_MULTI_LAYER:
		return downloadForMultiLayer(ag, task)
	default:
		err = errors.New("unsupportted task mode")
		return
	}

	return
}

func downloadForImage(ag *Agent, task *Task) (results []*DownloadResult, err error) {
	result, err := download(ag, task.ImageID, "image", task.URL)
	if err != nil {
		return
	}

	results = append(results, result)

	return
}

func downloadForLayer(ag *Agent, task *Task) (results []*DownloadResult, err error) {
	itemMap := make(map[string]*p2p.Item)

	for _, item := range task.Items {
		itemMap[item.ParentID] = item
	}

	id := ""
	for {
		item := itemMap[id]
		if item == nil {
			break
		}

		//TODO check if layer exists in docker using inspect API
		result, err := download(ag, item.ID, item.Type, item.URL)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
		id = item.ID
	}

	return
}

func downloadForMultiLayer(ag *Agent, task *Task) (results []*DownloadResult, err error) {
	var beginId string

	length := len(task.History)
	for i := length - 1; i >= 0; i-- {
		exist, err := p2p.LayerExist(ag.DockerClient, task.History[i])
		if err != nil {
			return nil, err
		}

		if !exist {
			beginId = task.History[i]
			break
		}
	}

	if len(beginId) != 0 {
		for _, item := range task.Items {
			if strings.HasSuffix(item.ID, beginId) {
				result, err := download(ag, item.ID, item.Type, item.URL)
				if err != nil {
					return nil, err
				}
				results = append(results, result)
				break
			}
		}
	}

	return
}

func download(ag *Agent, id, typee, url string) (result *DownloadResult, err error) {
	packageExist, packagePath, err := ag.PackageExist(id, typee)
	if err != nil {
		return
	}

	if packageExist {
		log.Printf("skip download package: %s", typee+"_"+id)

		result = &DownloadResult{
			Err:         nil,
			PackagePath: packagePath,
		}
		return
	}

torrent:
	torrentExist, torrentPath, err := ag.TorrentExist(id, typee)
	if err != nil {
		return
	}

	if !torrentExist {
		c, err := ag.PoolAdd("torrent_" + typee + "_" + id)
		if err != nil {
			if c != nil {
				log.Printf("torrent is being downloaded by other progress, please wait: %s", typee+"_"+id)
				<-c
				goto torrent
			} else {
				return nil, err
			}
		}

		log.Printf("++download torrent: %s", typee+"_"+id)

		if err = downloadTorrent(url, torrentPath); err != nil {
			return nil, err
		}
		log.Printf("--download torrent: %s", typee+"_"+id)

		ag.PoolDelete("torrent_" + typee + "_" + id)
	} else {
		log.Printf("skip download torrent: %s", typee+"_"+id)
	}

	log.Printf("++download package: %s", typee+"_"+id)
	c := make(chan error)
	configs := make(map[string]string)
	configs["target"] = "agent"
	go func(packagePath, torrentPath string, c chan error, typee, id string) {
		err = ag.BTClient.Download(packagePath, torrentPath, configs)
		if err != nil {
			os.Remove(packagePath)
		} else {
			log.Printf("--download package: %s", typee+"_"+id)
		}

		//TODO remove
		end := time.Now()
		Lock.Lock()
		if end.After(DownloadEnd) {
			DownloadEnd = end
		}
		Lock.Unlock()

		c <- err
	}(packagePath, torrentPath, c, typee, id)

	result = &DownloadResult{
		Err:         c,
		PackagePath: packagePath,
	}
	return
}

func downloadTorrent(url, path string) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("[ERROR]download torrent %s error: %s", url, resp.Status)
	}

	file, err := os.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	if _, err = io.Copy(file, resp.Body); err != nil {
		os.Remove(path)
		return
	}

	return
}
