package main

import (
	"fmt"
	"log"
	"net/http"
	"registry_p2p/manager"
	"time"
)

func registerHandler() {
	http.HandleFunc("/distribute", distributeHandler)
	http.Handle("/torrent/", http.FileServer(http.Dir(mg.DataDir)))
}

func distributeHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		endTime := time.Now()
		log.Printf("[statistics] %s %s %d", startTime.Format("2006-01-02T15:04:05"), endTime.Format("2006-01-02T15:04:05"), endTime.Unix()-startTime.Unix())
	}()

	task, err := manager.NewTask(r.Body, r.RemoteAddr, w)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if err := manager.Prepare(mg, task); err != nil {
		log.Println(err.Error())
		task.Writer.Write(err.Error())
		return
	}

	log.Printf("++distribte: %s", task.ImageName)
	task.Writer.Write(fmt.Sprintf("++distribte: %s", task.ImageName))

	if err := manager.Distribute(mg, task); err != nil {
		log.Printf("distribute error: %s", err.Error())
		task.Writer.Write(err.Error())
		return
	}

	log.Printf("--distribte: %s", task.ImageName)
	task.Writer.Write(fmt.Sprintf("--distribte: %s", task.ImageName))
}
