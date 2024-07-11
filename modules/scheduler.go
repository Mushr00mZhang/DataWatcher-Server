package modules

import (
	"github.com/robfig/cron/v3"
)

var (
	SchedulerStatusStop  int8 = 0
	SchedulerStatusStart int8 = 1
)

// 调度器
type Scheduler struct {
	Cron   *cron.Cron // Cron调度器
	Status int8       // 状态
}

func (scheduler *Scheduler) Init() {
	if scheduler.Cron == nil {
		scheduler.Cron = cron.New(
			cron.WithParser(
				cron.NewParser(cron.Second | cron.Minute | cron.Hour),
			),
		)
	}
}
func (scheduler *Scheduler) Start(watchers *[]*WatcherConfig, datasources *[]*Datasource, elastic *Elastic) {
	if scheduler.Status == SchedulerStatusStop {
		// fmt.Printf("GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
		scheduler.Status = SchedulerStatusStart
		scheduler.Init()
		for _, watcher := range *watchers {
			if watcher.EntryID != 0 {
				// watcher.Stop()
				continue
			}
			err := watcher.Start(scheduler.Cron, datasources, elastic)
			if err != nil {
				continue
			}
		}
		scheduler.Cron.Start()
	}
}

func (scheduler *Scheduler) Stop(watchers *[]*WatcherConfig) {
	scheduler.Cron.Stop()
	for _, watcher := range *watchers {
		watcher.Stop(scheduler.Cron)
	}
	scheduler.Status = SchedulerStatusStop
}
