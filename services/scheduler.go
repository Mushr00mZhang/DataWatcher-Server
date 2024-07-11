package services

import (
	"server/modules"
)

type SchedulerService struct {
	Watchers    *[]*modules.WatcherConfig
	Datasources *[]*modules.Datasource
	Scheduler   *modules.Scheduler
	Elastic     *modules.Elastic
}

func NewSchedulerService(watchers *[]*modules.WatcherConfig, datasources *[]*modules.Datasource, scheduler *modules.Scheduler, elastic *modules.Elastic) *SchedulerService {
	return &SchedulerService{
		Watchers:    watchers,
		Datasources: datasources,
		Scheduler:   scheduler,
		Elastic:     elastic,
	}
}

// 开启调度
func (service SchedulerService) Start() {
	service.Scheduler.Start(service.Watchers, service.Datasources, service.Elastic)
}

// 停止调度
func (service SchedulerService) Stop() {
	service.Scheduler.Stop(service.Watchers)
}
