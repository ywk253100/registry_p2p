package main

import (
	"log"
	"net/http"
	p2p "registry_p2p"
	"registry_p2p/agent"
	"time"
)

func registerHandler() {
	http.HandleFunc("/download", downloadHandler)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	totalStart := time.Now()

	task, err := agent.NewTask(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	//TODO id exists, name does not exist, just tag it
	imageExist, err := p2p.ImageExist(ag.DockerClient, task.ImageName, task.ImageID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err.Error())
		return
	}

	//TODO seed for others
	//image already exists
	if !imageExist {
		downloadStart := time.Now().Unix()
		results, err := agent.Download(ag, task)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Println(err.Error())
			return
		}

		loadStart, err := agent.Load(ag, task, results)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Println(err.Error())
			return
		}

		downloadEnd := agent.DownloadEnd

		loadEnd := time.Now().Unix()

		log.Printf("[statistics_download] %d %d %d", downloadStart, downloadEnd, downloadEnd-downloadStart)
		log.Printf("[statistics_load] %d %d %d", loadStart, loadEnd, loadEnd-loadStart)

	} else {
		log.Printf("image already exists: %s", task.ImageName)
	}

	totalEnd := time.Now()
	log.Printf("[statistics_success] %d %d %d", totalStart.Unix(), totalEnd.Unix(), totalEnd.Unix()-totalStart.Unix())

	return
}
