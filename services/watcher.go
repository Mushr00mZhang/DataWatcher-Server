package services

import (
	"errors"
	"fmt"
	"server/modules"
	"strings"

	"github.com/robfig/cron/v3"
)

var ErrWatcherNotFound = errors.New("watcher not found")

type WatcherService struct {
	Config      *modules.Config
	Watchers    *[]*modules.WatcherConfig
	Datasources *[]*modules.Datasource
	Scheduler   *modules.Scheduler
	Elastic     *modules.Elastic
}

func NewWatcherService(config *modules.Config, watchers *[]*modules.WatcherConfig, datasources *[]*modules.Datasource, scheduler *modules.Scheduler, elastic *modules.Elastic) *WatcherService {
	return &WatcherService{
		Config:      config,
		Watchers:    watchers,
		Datasources: datasources,
		Scheduler:   scheduler,
		Elastic:     elastic,
	}
}

// 获取监控列表
func (service *WatcherService) GetWatchers() *[]*modules.WatcherConfig {
	return service.Watchers
}

// 获取监控
func (service *WatcherService) GetWatcher(app string) (*modules.WatcherConfig, error) {
	for _, watcher := range *service.Watchers {
		if watcher.App == app {
			return watcher, nil
		}
	}
	return nil, ErrWatcherNotFound
}

// 创建监控
func (service *WatcherService) CreateWatcher(new *modules.WatcherConfig) error {
	service.Config.Mutex.Lock()
	defer service.Config.Mutex.Unlock()
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
	watchers := append(*service.Watchers, new)
	(*service.Watchers) = watchers
	service.Config.Save()
	return nil
}

// 更新监控
func (service *WatcherService) UpdateWatcher(app string, new *modules.WatcherConfig) error {
	service.Config.Mutex.Lock()
	defer service.Config.Mutex.Unlock()
	_, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	for i, watcher := range *service.Watchers {
		if watcher.App == app {
			switch {
			case !new.Enabled:
				watcher.Disable(service.Scheduler.Cron)
			case watcher.Cron != new.Cron:
				watcher.Stop(service.Scheduler.Cron)
			// case watcher.DataConfig.Type != new.DataConfig.Type:
			// 	watcher.Stop(service.Scheduler.Cron)
			// case watcher.DataConfig.DSN != new.DataConfig.DSN:
			// 	watcher.Stop(service.Scheduler.Cron)
			case watcher.GetExpired != new.GetExpired:
				watcher.Stop(service.Scheduler.Cron)
			}
			new.App = app
			new.EntryID = watcher.EntryID
			(*service.Watchers)[i] = new
			if new.Enabled {
				new.Start(service.Scheduler.Cron, service.Datasources, service.Elastic)
			}
			service.Config.Save()
		}
	}
	return nil
}

// 删除监控
func (service *WatcherService) DeleteWatcher(app string) error {
	service.Config.Mutex.Lock()
	defer service.Config.Mutex.Unlock()
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
		service.Config.Save()
		return nil
	}
}

// 启用监控
func (service *WatcherService) EnableWatcher(app string) error {
	service.Config.Mutex.Lock()
	defer service.Config.Mutex.Unlock()
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	err = watcher.Enable()
	if err != nil {
		return err
	}
	service.Config.Save()
	return nil
}

// 禁用监控
func (service *WatcherService) DisableWatcher(app string) error {
	service.Config.Mutex.Lock()
	defer service.Config.Mutex.Unlock()
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	watcher.Disable(service.Scheduler.Cron)
	service.Config.Save()
	return nil
}

// 开始监控
func (service *WatcherService) StartWatcher(app string) (cron.EntryID, error) {
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return 0, err
	}
	return watcher.Start(service.Scheduler.Cron, service.Datasources, service.Elastic)
}

// 停止监控
func (service *WatcherService) StopWatcher(app string) error {
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return err
	}
	watcher.Stop(service.Scheduler.Cron)
	return nil
}

// 获取监控状态
func (service *WatcherService) GetWatcherEntry(app string) (map[string]interface{}, error) {
	watcher, err := service.GetWatcher(app)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	if watcher.EntryID == 0 {
		res = map[string]interface{}{
			"App":  watcher.App,
			"ID":   0,
			"Prev": nil,
			"Next": nil,
		}
	} else {
		entry := service.Scheduler.Cron.Entry(watcher.EntryID)
		res = map[string]interface{}{
			"App":  watcher.App,
			"ID":   entry.ID,
			"Prev": entry.Prev,
			"Next": entry.Next,
		}
	}
	return res, nil
}

// 获取监控列表状态
func (service *WatcherService) GetEntries(apps []string) ([]map[string]interface{}, error) {
	var watchers []*modules.WatcherConfig
	if len(apps) == 0 {
		watchers = make([]*modules.WatcherConfig, len(apps))
		for i, app := range apps {
			watcher, err := service.GetWatcher(app)
			if err != nil {
				watchers[i] = nil
				fmt.Printf("Get watcher %s error : %s\n", app, err.Error())
			}
			watchers[i] = watcher
		}
	} else {
		watchers = *(service.GetWatchers())
	}
	res := make([]map[string]interface{}, len(watchers))
	for i, watcher := range watchers {
		if watcher == nil {
			res[i] = map[string]interface{}{
				"App":  "",
				"ID":   0,
				"Prev": nil,
				"Next": nil,
			}
		} else if watcher.EntryID == 0 {
			res[i] = map[string]interface{}{
				"App":  watcher.App,
				"ID":   0,
				"Prev": nil,
				"Next": nil,
			}
		} else {
			entry := service.Scheduler.Cron.Entry(watcher.EntryID)
			res[i] = map[string]interface{}{
				"App":  watcher.App,
				"ID":   entry.ID,
				"Prev": entry.Prev,
				"Next": entry.Next,
			}
		}
	}
	return res, nil
}
