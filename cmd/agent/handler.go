package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	p2p "p2p_lib"
	"p2p_lib/agent"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type tagId map[string]string
type repositories map[string]tagId

func registerHandler() {
	http.HandleFunc("/download", downloadHandler)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	//TODO remove it when complete tests
	startTime := time.Now()
	defer func() {
		endTime := time.Now()

		mark := "statistics_success"
		if err != nil {
			mark = "statistics_fail"
		}

		log.Printf("[%s] %s %s %d", mark, startTime.Format("2006-01-02T15:04:05"), endTime.Format("2006-01-02T15:04:05"), endTime.Unix()-startTime.Unix())
	}()

	var task *agent.DownloadTask

	defer func() {
		if err != nil {
			log.Println(err.Error())
		} else {
			log.Printf("complete download task: %s", task.Image)
		}
	}()

	log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.String())

	contentType := r.Header.Get("Content-Type")

	task, err = createDownloadTask(contentType, r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer os.RemoveAll(task.TmpDir)

	log.Printf("starting download task: %s", task.Image)

	var torrents []string
	for k, _ := range task.Torrents {
		torrents = append(torrents, k)
	}

	if err = p2p.Download(ag.BTClient, torrents); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	exist, err := p2p.ImageExist(ag.DockerClient, task.Image, task.ImageID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if exist {
		log.Printf("image already exists, seeding for others %s", task.Image)
		return
	}

	exist, _, err = ag.ImageTarExist(task.ImageID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if !exist {
		if err = agent.Assemble(task); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	if err = p2p.LoadImage(ag.DockerClient, task.ImageTarPath); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	return
}

func createDownloadTask(imageName, imageID, mode string) (task *agent.DownloadTask, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("create download task error: %s", err.Error())
		}
	}()

	task = &agent.DownloadTask{}

	json.Unmarshal()

	//	task.TmpDir = filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().UnixNano(), 10))
	//	if err = os.MkdirAll(task.TmpDir, 644); err != nil {
	//		return
	//	}

	//	task.Torrents = make(map[string]string)

	//	gr, err := gzip.NewReader(r)
	//	if err != nil {
	//		return
	//	}
	//	defer gr.Close()

	//	tr := tar.NewReader(gr)
	//	for {
	//		header, err := tr.Next()
	//		if err != nil {
	//			if err == io.EOF {
	//				break
	//			}
	//			return nil, err
	//		}

	//		switch header.Typeflag {
	//		case tar.TypeReg:
	//			if header.Name == "repositories" {
	//				repoPath := filepath.Join(task.TmpDir, header.Name)
	//				f, err := os.Create(repoPath)
	//				if err != nil {
	//					return nil, err
	//				}
	//				defer f.Close()

	//				if _, err := io.Copy(f, tr); err != nil {
	//					return nil, err
	//				}

	//				if _, err := f.Seek(0, 0); err != nil {
	//					return nil, err
	//				}

	//				data, err := ioutil.ReadAll(f)
	//				if err != nil {
	//					return nil, err
	//				}

	//				var repos repositories

	//				if err := json.Unmarshal(data, &repos); err != nil {
	//					return nil, err
	//				}

	//				for k, v := range repos {

	//					for k1, v1 := range v {
	//						task.Image = k + ":" + k1
	//						task.ImageID = v1
	//						break
	//					}
	//					break
	//				}
	//			} else {
	//				layerId := header.Name[:strings.Index(header.Name, ".torrent")]
	//				exist, layerTorrentPath, err := ag.LayerTorrentExist(layerId)
	//				if err != nil {
	//					return nil, err
	//				}

	//				if !exist {
	//					f, err := os.Create(layerTorrentPath)
	//					if err != nil {
	//						return nil, err
	//					}
	//					defer f.Close()

	//					if _, err := io.Copy(f, tr); err != nil {
	//						return nil, err
	//					}
	//				}

	//				task.Torrents[layerTorrentPath] = filepath.Join(ag.DataDir, "layer", layerId+".tar.gz")
	//			}

	//		default:
	//			return nil, errors.New("should not happen")
	//		}
	//	}

	//	task.ImageTarPath = filepath.Join(ag.DataDir, "image", task.ImageID+".tar")

	return task, err
}
