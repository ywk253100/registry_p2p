package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"p2p_lib/manager"
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
	listenAddr := flag.String("l", "0.0.0.0:8000", "listen-addr")
	dockerEndpoint := flag.String("e", "unix:///var/run/docker.sock", "docker daemon endpoint")
	btClient := flag.String("b", "anacrolix", "bt client(anacrolix, ctorrent)")
	scheduler := flag.String("s", "batch", "distribution scheduler")

	var ts trackers
	flag.Var(&ts, "t", "bt trackers, seperated by comma")

	flag.Parse()

	var err error
	mg, err = manager.CreateManager(*dataDir, *listenAddr, *dockerEndpoint, *btClient, *scheduler, ts)
	if err != nil {
		log.Fatal(err)
	}

	registerHandler()

	if err = http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
