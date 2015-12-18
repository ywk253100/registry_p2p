package manager

func Distribute(manager *Manager, task *Task) (err error) {
	err = manager.Scheduler.Schedule(task.ImageID, task.ImageName, task.Mode, task.URL, task.Items, task.Hosts)
	return
}
