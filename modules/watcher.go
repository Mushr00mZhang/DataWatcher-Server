package modules

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/robfig/cron/v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

var ErrWatcherNotFound = errors.New("watcher not found")

var Watchers *[]WatcherConfig
var (
	DataConfigTypeAPI       = "api"
	DataConfigTypeSQLServer = "sqlserver"
	DataConfigTypeMySQL     = "mysql"
	DataConfigTypeSQLite    = "sqlite"
	DataConfigTypeOracle    = "oracle"
)

// 监控数据配置
type WatcherDataConfig struct {
	Type       string `yaml:"Type"`       // 类型（api/sqlserver/mysql/sqlite）
	DSN        string `yaml:"DSN"`        // 连接串
	GetExpired string `yaml:"GetExpired"` // 获取过期数据
	GetTasks   string `yaml:"GetTasks"`   // 获取任务列表
	ResetTasks string `yaml:"ResetTasks"` // 重置任务状态
}

// 监控配置
type WatcherConfig struct {
	Module     string            `yaml:"Module"`     // 模块
	System     string            `yaml:"System"`     // 系统
	Provider   string            `yaml:"Provider"`   // 提供方
	Requester  string            `yaml:"Requester"`  // 请求方
	Type       string            `yaml:"Type"`       // 类型（Push/Pull）
	Method     string            `yaml:"Method"`     // 承载方式
	App        string            `yaml:"App"`        // 应用名称
	Desc       string            `yaml:"Desc"`       // 描述
	Interface  string            `yaml:"Interface"`  // 接口名称
	ConfigPath string            `yaml:"ConfigPath"` // 配置路径
	Tags       []string          `yaml:"Tags"`       // 标签
	DataConfig WatcherDataConfig `yaml:"DataConfig"` // 监控数据配置
	Extend     interface{}       `yaml:"Extend"`     // 扩展字段
	Cron       string            `yaml:"Cron"`       // Cron表达式
	Enabled    bool              `yaml:"Enabled"`    // 是否启用
	DB         *gorm.DB          `yaml:"-" json:"-"` // 连接池
	EntryID    cron.EntryID      `yaml:"-" json:"-"` // Cron运行时ID
	// Elastic    *Elastic          `yaml:"-" json:"-"` // ES
}

// 数据库连接
func (conf *WatcherConfig) Connect() error {
	if conf.DB != nil {
		return nil
	}
	if conf.DataConfig.DSN == "" {
		err := errors.New("DSN not found")
		// conf.Elastic.NewError("Connect to db failed", err.Error(), nil)
		return err
	}
	if conf.DataConfig.Type == "" {
		conf.DataConfig.Type = DataConfigTypeSQLServer
	}
	var open func(dsn string) gorm.Dialector
	switch conf.DataConfig.Type {
	case DataConfigTypeSQLServer:
		open = sqlserver.Open
	case DataConfigTypeMySQL:
		open = mysql.Open
	case DataConfigTypeSQLite:
		open = sqlite.Open
		// case DataConfigTypeOracle:
		// 	open = oracle.Open
	}
	if open != nil {
		db, err := gorm.Open(open(conf.DataConfig.DSN), &gorm.Config{})
		if err != nil {
			// conf.Elastic.NewError(fmt.Sprintf("Open %s failed", conf.DataConfig.Type), err.Error(), map[string]interface{}{
			// 	"Type": conf.DataConfig.Type,
			// 	"DSN":  conf.DataConfig.DSN,
			// })
			return err
		}
		conf.DB = db
	}
	return nil
}

// 从api获取数据
func (conf *WatcherConfig) GetExpiredDataFromAPI() (*ExpiredData, error) {
	url := conf.DataConfig.GetExpired
	resp, err := http.Get(url)
	if err != nil {
		// conf.Elastic.NewError("Get expired data failed", err.Error(), conf)
		return nil, err
	}
	defer resp.Body.Close()
	var data ExpiredData
	_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// conf.Elastic.NewError("Get expired data failed", err.Error(), nil)
		return nil, err
	}
	err = json.Unmarshal(_bytes, &data)
	if err != nil {
		// conf.Elastic.NewError("Get expired data failed", err.Error(), nil)
		return nil, err
	}
	return &data, nil
}

// 从数据库获取数据
func (conf *WatcherConfig) GetExpiredDataFromSQL() (*ExpiredData, error) {
	err := conf.Connect()
	if err != nil {
		// conf.Elastic.NewError("Get expired data failed", err.Error(), map[string]interface{}{
		// 	"DSN": conf.DataConfig.DSN,
		// })
		return nil, err
	}
	// 临时存储，获取所有平铺键值对，后续解析
	temp := map[string]interface{}{}
	conf.DB.Raw(conf.DataConfig.GetExpired).First(&temp)
	// 解析结果，将平铺键值对转为嵌套对象
	parsedInterface := parseInterface(temp)
	extend := parsedInterface["Extend"]
	data := ExpiredData{
		WatcherConfig: conf,
		TimeStamp:     time.Now().Local(),
		Expire1Day:    parseInt(parsedInterface["Expire1Day"]),
		Expire1Week:   parseInt(parsedInterface["Expire1Week"]),
		Expire1Month:  parseInt(parsedInterface["Expire1Month"]),
		Extend:        extend,
	}
	// TEST:
	// data.Expire1Day = rand.Intn(5)
	// data.Expire1Week = data.Expire1Week + rand.Intn(5)
	// data.Expire1Month = data.Expire1Week + rand.Intn(5)

	return &data, nil
}

func (conf *WatcherConfig) GetExpiredDataFunc(elastic *Elastic) func() {
	var getData func() (*ExpiredData, error)
	switch conf.DataConfig.Type {
	case DataConfigTypeAPI:
		getData = conf.GetExpiredDataFromAPI
	case DataConfigTypeSQLServer:
		getData = conf.GetExpiredDataFromSQL
	case DataConfigTypeMySQL:
		getData = conf.GetExpiredDataFromSQL
	case DataConfigTypeSQLite:
		getData = conf.GetExpiredDataFromSQL
	}
	if getData == nil {
		return nil
	}
	return func() {
		data, err := getData()
		if err != nil {
			return
		}
		elastic.Log(conf.App, data)
	}
}

// 解析为int
func parseInt(i interface{}) int {
	switch v := i.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case string:
		val, err := strconv.Atoi(v)
		if err == nil {
			return val
		}
	}
	return 0
}

// 将平铺键值对，递归解析为嵌套对象
func parseInterface(i map[string]interface{}) map[string]interface{} {
	obj := map[string]interface{}{}
	// 子集键值对
	subInterfaces := map[string]map[string]interface{}{}
	// 遍历所有键，直接赋值无“.”号键值对，记录带“.”号键值对
	for key := range i {
		if strings.Contains(key, ".") {
			// 记录带“.”号键值对
			kv := strings.Split(key, ".")
			subKey := kv[0]
			if subInterfaces[subKey] == nil {
				// 初始化
				subInterfaces[subKey] = map[string]interface{}{}
			}
			subInterface := subInterfaces[subKey]
			// 子键，剔除第一个字段，拼接后续字段
			subInterfaceKey := strings.Join(kv[1:], ".")
			subInterface[subInterfaceKey] = i[key]
		} else {
			// 接赋值无“.”号键值对
			obj[key] = i[key]
		}
	}
	for subKey := range subInterfaces {
		obj[subKey] = parseInterface(subInterfaces[subKey])
	}
	return obj
}

// 启用监控
func (conf *WatcherConfig) Enable() error {
	if conf.Enabled {
		return nil
	}
	conf.Enabled = true
	// conf.Start()
	return nil
}

// 启动监控
func (conf *WatcherConfig) Start(cron *cron.Cron, elastic *Elastic) error {
	if cron == nil {
		return nil
	}
	if !conf.Enabled {
		err := errors.New("watcher is disabled")
		// conf.Elastic.NewError("Start watcher failed", err.Error(), *conf)
		return err
	}
	if conf.EntryID != 0 {
		// err := errors.New("watcher is running")
		// conf.Elastic.NewError("Start watcher failed", err.Error(), *conf)
		// return err
		return nil
	}
	if conf.Cron == "" {
		err := errors.New("cron expression is null")
		// conf.Elastic.NewError("Start watcher failed", err.Error(), *conf)
		return err
	}
	fun := conf.GetExpiredDataFunc(elastic)
	id, err := cron.AddFunc(conf.Cron, fun)
	if err != nil {
		// conf.Elastic.NewError("Start watcher failed", err.Error(), *conf)
		return err
	}
	conf.EntryID = id
	return nil
}

// 禁用监控
func (conf *WatcherConfig) Disable(cron *cron.Cron) {
	if conf.DB != nil {
		conf.DB = nil
	}
	if cron != nil {
		conf.Stop(cron)
	}
}

// 关闭监控
func (conf *WatcherConfig) Stop(cron *cron.Cron) {
	if conf.DB != nil {
		conf.DB = nil
	}
	if conf.EntryID != 0 {
		if cron != nil {
			cron.Remove(conf.EntryID)
		}
		conf.EntryID = 0
	}
}

type ExpiredData struct {
	WatcherConfig *WatcherConfig ``                 // 配置
	TimeStamp     time.Time      `json:"timestamp"` // 时间戳
	Expire1Day    int            ``                 // 过期1天
	Expire1Week   int            ``                 // 过期7天
	Expire1Month  int            ``                 // 过期1个月
	Extend        interface{}    ``                 // 扩展字段
}

// // 上传到ES
// func (data *ExpiredData) ToES(elastic *Elastic) error {
// 	return elastic.Log(data.WatcherConfig.App, data)
// }

// // 获取监控列表
// func GetWatchers() *[]WatcherConfig {
// 	return Watchers
// }

// // 获取监控
// func GetWatcher(app string) (*WatcherConfig, error) {
// 	for _, watcher := range *Watchers {
// 		if watcher.App == app {
// 			return &watcher, nil
// 		}
// 	}
// 	return nil, ErrWatcherNotFound
// }

// // 创建监控
// func CreateWatcher(new WatcherConfig) error {
// 	if strings.TrimSpace(new.App) == "" {
// 		return errors.New("app is nil")
// 	}
// 	old, err := GetWatcher(new.App)
// 	if err != nil && err != ErrWatcherNotFound {
// 		return err
// 	}
// 	if old != nil {
// 		return errors.New("app is duplicated")
// 	}
// 	watchers := append(*Watchers, new)
// 	Watchers = &watchers
// 	return nil
// }

// // 更新监控
// func UpdateWatcher(app string, new WatcherConfig) error {
// 	_, err := GetWatcher(app)
// 	if err != nil {
// 		return err
// 	}
// 	// watchers := *Watchers
// 	for i, watcher := range *Watchers {
// 		if watcher.App == app {
// 			switch {
// 			case !new.Enabled:
// 				watcher.Disable()
// 			case watcher.Cron != new.Cron:
// 				watcher.Stop()
// 			case watcher.DataConfig.Type != new.DataConfig.Type:
// 				watcher.Stop()
// 			case watcher.DataConfig.DSN != new.DataConfig.DSN:
// 				watcher.Stop()
// 			case watcher.DataConfig.GetExpired != new.DataConfig.GetExpired:
// 				watcher.Stop()
// 			}
// 			new.App = app
// 			new.DB = watcher.DB
// 			new.EntryID = watcher.EntryID
// 			(*Watchers)[i] = new
// 			// Watchers = &watchers
// 			if new.Enabled {
// 				new.Start()
// 			}
// 		}
// 	}
// 	return nil
// }

// // 删除监控
// func DeleteWatcher(app string) error {
// 	l := len(*Watchers)
// 	i := 0
// 	watchers := *Watchers
// 	for _, watcher := range *Watchers {
// 		if watcher.App != app {
// 			watchers[i] = watcher
// 			i++
// 		} else {
// 			watcher.Disable()
// 		}
// 	}
// 	if l == i {
// 		return ErrWatcherNotFound
// 	} else {
// 		watchers = watchers[:i]
// 		Watchers = &watchers
// 		return nil
// 	}
// }

// // 启用监控
// func EnableWatcher(app string) error {
// 	watcher, err := GetWatcher(app)
// 	if err != nil {
// 		return err
// 	}
// 	return watcher.Enable()
// }

// // 禁用监控
// func DisableWatcher(app string) error {
// 	watcher, err := GetWatcher(app)
// 	if err != nil {
// 		return err
// 	}
// 	watcher.Disable()
// 	return nil
// }

// // 开始监控
// func StartWatcher(app string) error {
// 	watcher, err := GetWatcher(app)
// 	if err != nil {
// 		return err
// 	}
// 	return watcher.Start()
// }

// // 停止监控
// func StopWatcher(app string) error {
// 	watcher, err := GetWatcher(app)
// 	if err != nil {
// 		return err
// 	}
// 	watcher.Stop()
// 	return nil
// }

// func GetWatcherEntry(app string) (map[string]interface{}, error) {
// 	watcher, err := GetWatcher(app)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var res map[string]interface{}
// 	if watcher.EntryID == 0 {
// 		res = map[string]interface{}{
// 			"ID":   0,
// 			"Prev": nil,
// 			"Next": nil,
// 		}
// 	} else {
// 		entry := Cron.Entry(watcher.EntryID)
// 		res = map[string]interface{}{
// 			"ID":   entry.ID,
// 			"Prev": entry.Prev,
// 			"Next": entry.Next,
// 		}
// 	}
// 	return res, nil
// }
