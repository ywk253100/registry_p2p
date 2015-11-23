package scheduler

type PostData struct {
	ImageID   string
	ImageName string
	Mode      string
	URLs      []string
}

type Scheduler interface {
	Schedule(pd *PostData, hosts []string) error
}
