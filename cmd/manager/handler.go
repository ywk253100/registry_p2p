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
	start := time.Now()

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

	log.Printf("++distribute: %s", task.ImageName)
	task.Writer.Write(fmt.Sprintf("++distribute: %s \n", task.ImageName))

	if err := manager.Distribute(mg, task); err != nil {
		log.Printf("distribute error: %s", err.Error())
		task.Writer.Write(err.Error())
		return
	}

	log.Printf("--distribute: %s", task.ImageName)
	task.Writer.Write(fmt.Sprintf("--distribute: %s \n", task.ImageName))

	end := time.Now()
	log.Printf("[statistics] %d %d %f", start.Unix(), end.Unix(), end.Sub(start).Seconds())
	task.Writer.Write(fmt.Sprintf("[statistics] %d %d %f \n", start.Unix(), end.Unix(), end.Sub(start).Seconds()))
}
