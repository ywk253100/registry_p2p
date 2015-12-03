package agent

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	p2p "registry_p2p"
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

	exist, path, err := ag.ImageTarExist(task.ImageID)
	if err != nil {
		return
	}

	if exist {
		//TODO if exists, seed for others
		log.Printf("image tar already exists: %s", task.ImageName)
		return
	}

	log.Printf("++download: %s", task.ImageName)
	packagePaths, err := download(ag, task.Items)
	if err != nil {
		return
	}
	log.Printf("--download: %s", task.ImageName)

	log.Printf("++create image tar: %s", task.ImageName)
	if task.Mode == "image" {
		//decompress
		packageFile, err := os.Open(packagePaths[0])
		if err != nil {
			return "", err
		}
		defer packageFile.Close()

		gr, err := gzip.NewReader(packageFile)
		if err != nil {
			return "", err
		}
		defer gr.Close()

		imageTarFile, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer imageTarFile.Close()

		if _, err = io.Copy(imageTarFile, gr); err != nil {
			os.Remove(path)
			return "", err
		}

	} else {
		if err = Assemble(packagePaths, path); err != nil {
			return "", err
		}
	}
	log.Printf("--create image tar: %s", task.ImageName)

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
		}

		packageExist, packagePath, err := ag.PackageExist(item.ID, item.Type)
		if err != nil {
			return nil, err
		}
		paths = append(paths, packagePath)

		if packageExist {
			continue
		}

		c := make(chan error)
		result[item.URL] = c
		configs := make(map[string]string)
		configs["target"] = "agent"
		go func(packagePath, torrentPath string, c chan error) {
			err = ag.BTClient.Download(packagePath, torrentPath, configs)
			c <- err
			if err != nil {
				os.Remove(packagePath)
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
