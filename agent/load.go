package agent

import (
	"bufio"
	"log"
	"os"
	p2p "registry_p2p"
)

func Load(ag *Agent, results []*DownloadResult) (err error) {
	bufSize := 1024 * 1024 * 10

	for _, result := range results {
		if result.Err != nil {
			err := <-result.Err
			if err != nil {
				return err
			}
		}

		log.Printf("++load: %s", result.PackagePath)
		file, err := os.Open(result.PackagePath)
		if err != nil {
			return err
		}
		defer file.Close()

		r := bufio.NewReaderSize(file, bufSize)

		if err = p2p.LoadImage(ag.DockerClient, r); err != nil {
			return err
		}
		log.Printf("--load: %s", result.PackagePath)
	}

	return
}
