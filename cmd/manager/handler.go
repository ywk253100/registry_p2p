package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"p2p_lib"
	"p2p_lib/manager"
	"p2p_lib/manager/scheduler"
	"strings"
	"time"
)

var (
	ImageNIL error = errors.New("image is nil")
)

func registerHandler() {
	http.HandleFunc("/distribute", distributeHandler)
	http.Handle("/", http.FileServer(http.Dir(mg.DataDir)))
}

func distributeHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	//TODO remove it when complete tests
	startTime := time.Now()
	defer func() {
		endTime := time.Now()
		log.Printf("[statistics] %s %s %d", startTime.Format("2006-01-02T15:04:05"), endTime.Format("2006-01-02T15:04:05"), endTime.Unix()-startTime.Unix())
	}()

	defer func() {
		if err != nil {
			statusCode := 500
			if err == ImageNIL {
				statusCode = 400
			}
			http.Error(w, err.Error(), statusCode)
			log.Printf("[ERROR] %s %s", r.RemoteAddr, err.Error())
		}
	}()

	task, err := createDistributionTask(r.Body, w, r.RemoteAddr)
	if err != nil {
		return
	}

	task.Client.Write("distribution starting...")
	log.Printf("++++distribue: %s", task.ImageName)

	if err := manager.Prepare(mg, task); err != nil {
		statusCode := 500
		if strings.Contains(err.Error(), "Authentication is required") {
			statusCode = 401
		}
		http.Error(w, err.Error(), statusCode)
		log.Println(err.Error())
		return
	}

	configs := make(map[string]string)
	configs["target"] = "manager"
	if err := p2p_lib.Download(mg.BTClient, task.Torrents, configs); err != nil {
		log.Printf("seed $s error: %s", task.ImageName, err.Error())
		return
	}

	if err := manager.Distribute(mg, task); err != nil {
		log.Printf("distribute $s error: %s", task.ImageName, err.Error())
		return
	}

	log.Printf("--------distribute: %s", task.ImageName)
}

func createDistributionTask(r io.Reader, w io.Writer, source string) (task *manager.DistributionTask, err error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}

	task = &manager.DistributionTask{}

	if err = json.Unmarshal(data, task); err != nil {
		return
	}

	if len(task.ImageName) == 0 {
		err = ImageNIL
		return
	}

	if len(task.Mode) == 0 {
		task.Mode = p2p_lib.ImageMode
	}

	task.Client = &manager.Client{
		W: w,
	}
	task.Source = source
	task.Torrents = make(map[string]string)

	task.PD = &scheduler.PostData{
		ImageName: task.ImageName,
		Mode:      task.Mode,
	}

	return
}
