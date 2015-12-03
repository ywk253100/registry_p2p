package main

import (
	"log"
	"os"
	//"log"
	"net/http"
	p2p "registry_p2p"
	"registry_p2p/agent"
)

type tagId map[string]string
type repositories map[string]tagId

func registerHandler() {
	http.HandleFunc("/download", downloadHandler)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	task, err := agent.NewTask(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	//TODO id exists, name does not exist, just tag it
	exist, err := p2p.ImageExist(ag.DockerClient, task.ImageName, task.ImageID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err.Error())
		return
	}

	//image already exists
	if exist {
		log.Printf("image already exists: %s", task.ImageName)
		return
	}

	path, err := agent.Download(ag, task)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err.Error())
		return
	}

	file, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err.Error())
		return
	}
	defer file.Close()

	log.Printf("++load image: %s", task.ImageName)
	if err = p2p.LoadImage(ag.DockerClient, task.ImageName, file); err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err.Error())
		return
	}
	log.Printf("--load image: %s", task.ImageName)

	return
}
