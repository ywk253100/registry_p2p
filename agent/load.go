package agent

import (
	"bufio"
	"log"
	"os"
	p2p "registry_p2p"
	"strings"
	"time"
)

func Load(ag *Agent, task *Task, results []*DownloadResult) (loadStart time.Time, err error) {
	var flag time.Time
	bufSize := 1024 * 1024 * 10

	for _, result := range results {
		if result.Err != nil {
			err := <-result.Err
			if err != nil {
				return loadStart, err
			}
		}

		//TODO remove
		if loadStart.Equal(flag) {
			loadStart = time.Now()
		}

		log.Printf("++load: %s", result.PackagePath)
		file, err := os.Open(result.PackagePath)
		if err != nil {
			return loadStart, err
		}
		defer file.Close()

		r := bufio.NewReaderSize(file, bufSize)

		if err = p2p.LoadImage(ag.DockerClient, r); err != nil {
			return loadStart, err
		}
		log.Printf("--load: %s", result.PackagePath)
	}

	if task.Mode == p2p.MODE_IMAGE {
		return
	}

	log.Printf("++tag: %s", task.ImageName)

	repo := task.ImageName
	tag := "latest"

	index := strings.LastIndex(task.ImageName, ":")
	if index != -1 {
		repo = task.ImageName[:index]
		tag = task.ImageName[index+1:]
	}

	if err = p2p.Tag(ag.DockerClient, task.ImageID, repo, tag); err != nil {
		return
	}

	log.Printf("--tag: %s", task.ImageName)

	return
}
