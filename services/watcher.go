package services

import (
	"errors"
	"server/modules"
	"strings"
)

var ErrWatcherNotFound = errors.New("watcher not found")

type WatcherService struct {
	Watchers  *[]*modules.WatcherConfig
	Scheduler *modules.Scheduler
	Elastic   *modules.Elastic
}

func NewWatcherService(watchers *[]*modules.WatcherConfig, scheduler *modules.Scheduler, elastic *modules.Elastic) *WatcherService {
	return &WatcherService{
		Watchers:  watchers,
		Scheduler: scheduler,
		Elastic:   elastic,
	}
}

// 获取监控列表
func (service WatcherService) GetWatchers() *[]*modules.WatcherConfig {
	return service.Watchers
}

// 获取监控
func (service WatcherService) GetWatcher(app string) (*modules.WatcherConfig, error) {
	for _, watcher := range *service.Watchers {
		if watcher.App == app {
			return watcher, nil
		}
	}
	return nil, ErrWatcherNotFound
}

// 创建监控
func (service WatcherService) CreateWatcher(new modules.WatcherConfig) error {
	if strings.TrimSpace(new.App) == "" {
		return errors.New("app is nil")
	}
	old, err := service.GetWatcher(new.App)
	if err != nil && err != ErrWatcherNotFound {
		return err
	}
	if old != nil {
		return errors.New("app is duplicated")
	}
	watchers := append(*service.Watchers, &new)
	service.Watchers = &watchers
	return nil
}

// 更新监控
func (service WatcherService) UpdateWatcher(app string, new modules.WatcherConfig) error {
	_, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	// watchers := *Watchers
	for i, watcher := range *service.Watchers {
		if watcher.App == app {
			switch {
			case !new.Enabled:
				watcher.Disable(service.Scheduler.Cron)
			case watcher.Cron != new.Cron:
				watcher.Stop(service.Scheduler.Cron)
			case watcher.DataConfig.Type != new.DataConfig.Type:
				watcher.Stop(service.Scheduler.Cron)
			case watcher.DataConfig.DSN != new.DataConfig.DSN:
				watcher.Stop(service.Scheduler.Cron)
			case watcher.DataConfig.GetExpired != new.DataConfig.GetExpired:
				watcher.Stop(service.Scheduler.Cron)
			}
			new.App = app
			new.DB = watcher.DB
			new.EntryID = watcher.EntryID
			(*service.Watchers)[i] = &new
			// Watchers = &watchers
			if new.Enabled {
				new.Start(service.Scheduler.Cron, service.Elastic)
			}
		}
	}
	return nil
}

// 删除监控
func (service WatcherService) DeleteWatcher(app string) error {
	l := len(*service.Watchers)
	i := 0
	watchers := *service.Watchers
	for _, watcher := range *service.Watchers {
		if watcher.App != app {
			(*service.Watchers)[i] = watcher
			i++
		} else {
			watcher.Disable(service.Scheduler.Cron)
		}
	}
	if l == i {
		return ErrWatcherNotFound
	} else {
		(*service.Watchers) = watchers[:i]
		// service.Watchers = &watchers
		return nil
	}
}

// 启用监控
func (service WatcherService) EnableWatcher(app string) error {
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	return watcher.Enable()
}

// 禁用监控
func (service WatcherService) DisableWatcher(app string) error {
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	watcher.Disable(service.Scheduler.Cron)
	return nil
}

// 开始监控
func (service WatcherService) StartWatcher(app string) error {
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	return watcher.Start(service.Scheduler.Cron, service.Elastic)
}

// 停止监控
func (service WatcherService) StopWatcher(app string) error {
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	watcher.Stop(service.Scheduler.Cron)
	return nil
}

// 获取监控Id
func (service WatcherService) GetWatcherEntry(app string) (map[string]interface{}, error) {
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	if watcher.EntryID == 0 {
		res = map[string]interface{}{
			"ID":   0,
			"Prev": nil,
			"Next": nil,
		}
	} else {
		entry := service.Scheduler.Cron.Entry(watcher.EntryID)
		res = map[string]interface{}{
			"ID":   entry.ID,
			"Prev": entry.Prev,
			"Next": entry.Next,
		}
	}
	return res, nil
}
