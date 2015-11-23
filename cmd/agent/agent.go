package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"p2p_lib/agent"
)

//TODO remove file or dir when error occur

var ag *agent.Agent

func main() {
	dataDir := flag.String("d", "/p2p/", "data directory")
	listenAddr := flag.String("l", "0.0.0.0:8000", "listen-addr")
	dockerEndpoint := flag.String("e", "unix:///var/run/docker.sock", "docker daemon endpoint")
	btClient := flag.String("b", "anacrolix", "bt client(anacrolix, ctorrent)")

	flag.Parse()

	var err error
	ag, err = agent.CreateAgent(*dataDir, *listenAddr, *dockerEndpoint, *btClient)
	if err != nil {
		log.Fatal(err)
	}

	//TODO remove it when complete tests
	if len(os.Getenv("agent_port")) != 0 {
		*listenAddr = "0.0.0.0:" + os.Getenv("agent_port")
	}

	registerHandler()

	log.Printf("agent is listening on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
