package manager

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	p2p "registry_p2p"
	"registry_p2p/utils"
)

type Task struct {
	ImageID string

	ImageName string   `json:"image"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Email     string   `json:"email"`
	Mode      string   `json:"mode"`
	Hosts     []string `json:"hosts"`

	Owner string

	State       string
	AgentStates map[string]string
	Items       []*p2p.Item

	Writer *utils.FlushWriter
}

func NewTask(r io.Reader, owner string, w io.Writer) (task *Task, err error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}

	task = &Task{}

	if err = json.Unmarshal(data, task); err != nil {
		return
	}

	if len(task.ImageName) == 0 {
		err = errors.New("image is null")
		return
	}

	if len(task.Mode) == 0 {
		task.Mode = p2p.MODE_IMAGE
	}

	task.Writer = utils.NewFlushWriter(w)

	return
}
