package scheduler

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	p2p "registry_p2p"
	"strings"
	"sync"
)

const (
	ratio int = 2
)

type Batch struct {
}

func NewBatch() (batch *Batch) {
	batch = &Batch{}
	return
}

func (b *Batch) Schedule(imageID, imageName, mode string, items []*p2p.Item, hosts []string) (err error) {
	image := &Image{
		ID:    imageID,
		Name:  imageName,
		Mode:  mode,
		Items: items,
	}

	data, err := json.Marshal(image)
	if err != nil {
		return
	}

	cond := sync.NewCond(new(sync.Mutex))

	var readys []chan bool
	for _, host := range hosts {
		ready := make(chan bool)
		readys = append(readys, ready)
		go post(host, bytes.NewReader(data), cond, ratio, ready)
	}

	for _, ready := range readys {
		<-ready
	}

	return
}

//TODO return after sending request immediately
func post(url string, payload io.Reader, cond *sync.Cond, ratio int, ready chan bool) {
	cond.L.Lock()
	ready <- true
	cond.Wait()
	cond.L.Unlock()

	log.Printf("++++post: %s", url)

	u := url

	if !strings.HasPrefix(u, "http://") {
		u = "http://" + u
	}

	if !strings.HasSuffix(u, "/") {
		u = u + "/"
	}

	u = u + "download"

	resp, err := http.Post(url, "application/json", payload)
	if err != nil {
		log.Printf("post to %s error: %s", url, err.Error())
		return
	}

	if resp.StatusCode != 200 {
		log.Printf("post to %s error, status code %d", url, resp.StatusCode)
		return
	}

	log.Printf("--------post: %s", url)

	for i := 0; i < ratio; i++ {
		cond.L.Lock()
		cond.Signal()
		cond.L.Unlock()
	}
}
