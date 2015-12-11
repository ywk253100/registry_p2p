package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"registry_p2p/agent"
)

//TODO remove file or dir when error occur

var ag *agent.Agent

func main() {
	dataDir := flag.String("d", "/p2p/", "data directory")
	port := flag.String("p", "8000", "port")
	dockerEndpoint := flag.String("e", "unix:///var/run/docker.sock", "docker daemon endpoint")
	btClient := flag.String("b", "ctorrent", "bt client(builtin, ctorrent)")

	flag.Parse()

	var err error
	ag, err = agent.NewAgent(*dataDir, *port, *dockerEndpoint, *btClient)
	if err != nil {
		log.Fatal(err)
	}

	//TODO remove it when complete tests
	if len(os.Getenv("agent_port")) != 0 {
		*port = os.Getenv("agent_port")
	}

	registerHandler()

	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal(err)
	}
}
