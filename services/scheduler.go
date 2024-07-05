package services

import (
	"server/modules"
)

type SchedulerService struct {
	Watchers  *[]*modules.WatcherConfig
	Scheduler *modules.Scheduler
	Elastic   *modules.Elastic
}

func NewSchedulerService(watchers *[]*modules.WatcherConfig, scheduler *modules.Scheduler, elastic *modules.Elastic) *SchedulerService {
	return &SchedulerService{
		Watchers:  watchers,
		Scheduler: scheduler,
		Elastic:   elastic,
	}
}

// 开启调度
func (service SchedulerService) Start() {
	service.Scheduler.Start(service.Watchers, service.Elastic)
	// if service.Scheduler.Status == modules.SchedulerStatusStop {
	// 	// fmt.Printf("GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
	// 	service.Scheduler.Status = modules.SchedulerStatusStart
	// 	if service.Scheduler.Cron == nil {
	// 		service.Scheduler.Cron = cron.New(
	// 			cron.WithParser(
	// 				cron.NewParser(cron.Second | cron.Minute | cron.Hour),
	// 			),
	// 		)
	// 		watcher.Cron = service.Scheduler.Cron
	// 	}
	// 	for _, watcher := range *service.Watchers {
	// 		if watcher.EntryID != 0 {
	// 			// watcher.Stop()
	// 			continue
	// 		}
	// 		err := watcher.Start(service.Scheduler.Cron, service.Elastic)
	// 		if err != nil {
	// 			continue
	// 		}
	// 	}
	// 	service.Scheduler.Cron.Start()
	// }
}

// 停止调度
func (service SchedulerService) Stop() {
	service.Scheduler.Stop(service.Watchers)
}
