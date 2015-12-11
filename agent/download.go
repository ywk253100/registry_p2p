package agent

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	p2p "registry_p2p"
	"time"
)

func Download(ag *Agent, task *Task) (path string, err error) {
image:
	c, err := ag.PoolAdd(task.Mode + "_" + task.ImageName)
	if err != nil {
		if c != nil {
			log.Printf("task is already in progress: %s", task.Mode+"_"+task.ImageName)
			<-c
			goto image
		} else {
			return
		}
	}
	defer ag.PoolDelete(task.Mode + "_" + task.ImageName)

	start := time.Now()
	log.Printf("++download: %s", task.ImageName)
	packagePaths, err := download(ag, task.Items)
	if err != nil {
		return
	}
	log.Printf("--download: %s", task.ImageName)
	end := time.Now()
	log.Printf("[statistics_download] %d %d %f", start.Unix(), end.Unix(), end.Sub(start).Seconds())

	if task.Mode == p2p.MODE_LAYER {
		start := time.Now()
		log.Printf("++assemble: %s", task.ImageName)
		_, path, err = ag.ImageTarExist(task.ImageID)
		if err != nil {
			return
		}
		if err = Assemble(packagePaths, path); err != nil {
			return "", err
		}
		log.Printf("--assemble: %s", task.ImageName)
		end := time.Now()
		log.Printf("[statistics_download] %d %d %f", start.Unix(), end.Unix(), end.Sub(start).Seconds())
	} else {
		path = packagePaths[0]
	}

	return
}

func download(ag *Agent, items []*p2p.Item) (paths []string, err error) {
	result := make(map[string]chan error)

	for _, item := range items {
	torrent:
		torrentExist, torrentPath, err := ag.TorrentExist(item.ID, item.Type)
		if err != nil {
			return nil, err
		}

		if !torrentExist {
			c, err := ag.PoolAdd("torrent_" + item.Type + "_" + item.ID)
			if err != nil {
				if c != nil {
					<-c
					goto torrent
				} else {
					return nil, err
				}
			}

			log.Printf("++download torrent: %s", item.URL)
			if err = downloadTorrent(item.URL, torrentPath); err != nil {
				return nil, err
			}
			log.Printf("--download torrent: %s", item.URL)

			ag.PoolDelete("torrent_" + item.Type + "_" + item.ID)
		} else {
			log.Printf("torrent already exist: %s", item.URL)
		}

		_, packagePath, err := ag.PackageExist(item.ID, item.Type)
		if err != nil {
			return nil, err
		}
		paths = append(paths, packagePath)

		//		if packageExist {
		//			continue
		//		}

		log.Printf("++download package: %s", packagePath)
		c := make(chan error)
		result[item.URL] = c
		configs := make(map[string]string)
		configs["target"] = "agent"
		go func(packagePath, torrentPath string, c chan error) {
			err = ag.BTClient.Download(packagePath, torrentPath, configs)
			c <- err
			if err != nil {
				os.Remove(packagePath)
			} else {
				log.Printf("--download package: %s", packagePath)
			}
		}(packagePath, torrentPath, c)
	}

	for url, c := range result {
		if e := <-c; e != nil {
			return nil, fmt.Errorf("[ERROR]download %s error: %s", url, e.Error())
		}
		delete(result, url)
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
