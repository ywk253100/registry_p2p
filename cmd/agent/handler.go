package main

import (
	"log"
	"net/http"
	p2p "registry_p2p"
	"registry_p2p/agent"
	"time"
)

type tagId map[string]string
type repositories map[string]tagId

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
	if imageExist {
		log.Printf("image already exists: %s", task.ImageName)
		return
	}

	var path string

	if task.Mode == p2p.MODE_LAYER {
		imageTarExist, imageTarPath, err := ag.ImageTarExist(task.ImageID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Println(err.Error())
			return
		}

		if imageTarExist {
			path = imageTarPath
			goto load
		}
	}

	path, err = agent.Download(ag, task)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err.Error())
		return
	}

load:
	start := time.Now()
	log.Printf("++load image: %s", task.ImageName)
	if err = agent.Load(ag.DockerClient, path, task.Mode); err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err.Error())
		return
	}
	log.Printf("--load image: %s", task.ImageName)
	end := time.Now()

	log.Printf("[statistics_load] %d %d %f", start.Unix(), end.Unix(), end.Sub(start).Seconds())

	totalEnd := time.Now()
	log.Printf("[statistics_success] %d %d %f", totalStart.Unix(), totalEnd.Unix(), totalEnd.Sub(totalStart).Seconds())

	return
}
