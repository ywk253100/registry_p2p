package scheduler

import (
	p2p "registry_p2p"
)

type Image struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Mode  string      `json:"mode"`
	Items []*p2p.Item `json:"items"`
	URL   string      `json:"url"`
}

type Scheduler interface {
	Schedule(imageID, imageName, mode, url string, items []*p2p.Item, hosts []string) error
}
