package manager

func Distribute(manager *Manager, task *DistributionTask) (err error) {
	err = manager.Scheduler.Schedule(task.PD, task.Hosts)
	return
}
