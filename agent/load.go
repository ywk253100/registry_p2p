package agent

import (
	"bufio"
	"log"
	"os"
	p2p "registry_p2p"
	"strings"
	"time"
)

func Load(ag *Agent, task *Task, results []*DownloadResult) (loadStart int64, err error) {
	loadStart = 0
	bufSize := 1024 * 1024 * 10

	for _, result := range results {
		if result.Err != nil {
			err := <-result.Err
			if err != nil {
				return 0, err
			}
		}

		//TODO remove
		if loadStart == 0 {
			loadStart = time.Now().Unix()
		}

		log.Printf("++load: %s", result.PackagePath)
		file, err := os.Open(result.PackagePath)
		if err != nil {
			return 0, err
		}
		defer file.Close()

		r := bufio.NewReaderSize(file, bufSize)

		if err = p2p.LoadImage(ag.DockerClient, r); err != nil {
			return 0, err
		}
		log.Printf("--load: %s", result.PackagePath)
	}

	if task.Mode == p2p.MODE_IMAGE {
		return
	}

	log.Printf("++tag: %s", task.ImageName)

	strs := strings.Split(task.ImageName, ":")
	repo := strs[0]
	tag := ""
	if len(strs[1]) != 0 {
		tag = strs[1]
	}

	if err = p2p.Tag(ag.DockerClient, task.ImageID, repo, tag); err != nil {
		return
	}

	log.Printf("--tag: %s", task.ImageName)

	return
}
