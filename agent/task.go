package agent

import (
	p2p "registry_p2p"
)

import (
	"encoding/json"
	"io"
	"io/ioutil"
)

type Task struct {
	ImageID   string      `json:"id"`
	ImageName string      `json:"name"`
	Mode      string      `json:"mode"`
	URL       string      `json:"url"`
	Items     []*p2p.Item `json:"items"`
	History   []string    `json:"history"`
}

func NewTask(r io.Reader) (task *Task, err error) {

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}

	task = &Task{}

	if err = json.Unmarshal(b, task); err != nil {
		return
	}

	return
}
