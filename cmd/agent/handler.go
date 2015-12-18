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
		//downloadStart := time.Now()
		results, err := agent.Download(ag, task)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Println(err.Error())
			return
		}
		//downloadEnd := time.Now()
		//log.Printf("[statistics_download] %d %d %f", downloadStart.Unix(), downloadEnd.Unix(), downloadEnd.Sub(downloadStart).Seconds())

		//loadStart := time.Now()

		if err = agent.Load(ag, results); err != nil {
			http.Error(w, err.Error(), 500)
			log.Println(err.Error())
			return
		}

		//loadEnd := time.Now()

		//log.Printf("[statistics_load] %d %d %f", loadStart.Unix(), loadEnd.Unix(), loadEnd.Sub(loadStart).Seconds())

	} else {
		log.Printf("image already exists: %s", task.ImageName)
	}

	totalEnd := time.Now()
	log.Printf("[statistics_success] %d %d %f", totalStart.Unix(), totalEnd.Unix(), totalEnd.Sub(totalStart).Seconds())

	return
}
