package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"registry_p2p/manager"
	"strings"
)

var mg *manager.Manager

type trackers []string

func (t *trackers) String() string {
	return fmt.Sprint(*t)
}

func (t *trackers) Set(value string) error {
	if len(*t) > 0 {
		return errors.New("trackers flag already set.")
	}
	for _, tracker := range strings.Split(value, ",") {
		*t = append(*t, tracker)
	}
	return nil
}

func main() {
	dataDir := flag.String("d", "/p2p/", "data directory")
	port := flag.String("p", "8000", "port")
	dockerEndpoint := flag.String("e", "unix:///var/run/docker.sock", "docker daemon endpoint")
	btClient := flag.String("b", "anacrolix", "bt client(anacrolix, ctorrent)")
	scheduler := flag.String("s", "batch", "distribution scheduler")

	var ts trackers
	flag.Var(&ts, "t", "bt trackers, seperated by comma")

	flag.Parse()

	var err error
	mg, err = manager.NewManager(*dataDir, *port, *dockerEndpoint, *btClient, *scheduler, ts)
	if err != nil {
		log.Fatal(err)
	}

	registerHandler()

	if err = http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal(err)
	}
}
