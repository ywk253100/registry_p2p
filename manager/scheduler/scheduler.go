package scheduler

import (
	p2p "registry_p2p"
)

type Image struct {
	ID    string
	Name  string
	Mode  string
	Items []*p2p.Item
}

type Scheduler interface {
	Schedule(imageID, imageName, mode string, items []*p2p.Item, hosts []string) error
}
