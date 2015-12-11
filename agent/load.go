package agent

import (
	"bufio"
	"compress/gzip"
	docker "github.com/fsouza/go-dockerclient"
	"io"
	//"io"
	"os"
	p2p "registry_p2p"
)

func Load(client *docker.Client, path, mode string) (err error) {
	bufSize := 1024 * 1024
	var r io.Reader

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	r = bufio.NewReaderSize(file, bufSize)

	if mode == p2p.MODE_IMAGE {
		r, err = gzip.NewReader(r)
		if err != nil {
			return err
		}
	}

	if err = p2p.LoadImage(client, r); err != nil {
		return
	}

	return
}
