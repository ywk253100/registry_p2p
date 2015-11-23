package scheduler

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
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

func (b *Batch) Schedule(pd *PostData, hosts []string) (err error) {
	data, err := json.Marshal(pd)
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

	//	var b bytes.Buffer
	//	w := multipart.NewWriter(&b)

	//	if err := w.WriteField("id", imageId); err != nil {
	//		log.Printf("post to %s error: %s", url, err.Error())
	//		return
	//	}

	//	if err := w.WriteField("name", imageName); err != nil {
	//		log.Printf("post to %s error: %s", url, err.Error())
	//		return
	//	}

	//	if err := w.WriteField("mode", mode); err != nil {
	//		log.Printf("post to %s error: %s", url, err.Error())
	//		return
	//	}

	//	ww, err := w.CreateFormField("torrent")
	//	if err != nil {
	//		log.Printf("post to %s error: %s", url, err.Error())
	//		return
	//	}

	//	if _, err := io.Copy(ww, payload); err != nil {
	//		log.Printf("post to %s error: %s", url, err.Error())
	//		return
	//	}

	//	req, err := http.NewRequest("POST", u, &b)
	//	if err != nil {
	//		log.Printf("post to %s error: %s", url, err.Error())
	//		return
	//	}

	//	req.Header.Set("Content-Type", w.FormDataContentType())

	//	client := &http.Client{}

	//	resp, err := client.Do(req)
	//	if err != nil {
	//		log.Printf("post to %s error: %s", url, err.Error())
	//		return
	//	}

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
