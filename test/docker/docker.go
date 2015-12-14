package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {
	var listenAddr string
	if len(os.Getenv("agent_port")) != 0 {
		listenAddr = "0.0.0.0:" + os.Getenv("agent_port")
	}

	http.HandleFunc("/download", download)
	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Println(err.Error())
		return
	}
}

func download(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("bash", "-c", "docker.sh")
	if err := cmd.Run(); err != nil {
		log.Println(err.Error())
		return
	}
}
