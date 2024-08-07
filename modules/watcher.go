package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
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
	Database   string `yaml:"Database"`   // 数据库
	GetExpired string `yaml:"GetExpired"` // 获取过期数据
}

// 监控配置
type WatcherConfig struct {
	Mutex        sync.Mutex   `yaml:"-" json:"-"` // 互斥锁
	Module       string       `yaml:"Module"`     // 模块
	System       string       `yaml:"System"`     // 系统
	Provider     string       `yaml:"Provider"`   // 提供方
	Requester    string       `yaml:"Requester"`  // 请求方
	Type         string       `yaml:"Type"`       // 类型（Push/Pull）
	Method       string       `yaml:"Method"`     // 承载方式
	App          string       `yaml:"App"`        // 应用名称
	Desc         string       `yaml:"Desc"`       // 描述
	Interface    string       `yaml:"Interface"`  // 接口名称
	ConfigPath   string       `yaml:"ConfigPath"` // 配置路径
	Tags         []string     `yaml:"Tags"`       // 标签
	Sources      []string     `yaml:"Sources"`    // 数据源编号列表
	GetExpired   string       `yaml:"GetExpired"` // 获取呆滞数据SQL
	Extend       interface{}  `yaml:"Extend"`     // 扩展字段
	Cron         string       `yaml:"Cron"`       // Cron表达式
	Enabled      bool         `yaml:"Enabled"`    // 是否启用
	EntryID      cron.EntryID `yaml:"-"`          // Cron运行时ID
	Count        int64        `yaml:"-"`          // 运行次数
	PrevDuration int64        `yaml:"-"`          // 上次运行耗时(ms)
	DurationAvg  int64        `yaml:"-"`          // 运行平均耗时(ms)
}

// 从api获取数据
func (watcher *WatcherConfig) GetExpiredDataFromAPI(datasource *Datasource) (*[]*ExpiredData, error) {
	resp, err := http.Get(datasource.Url)
	if err != nil {
		// watcher.Elastic.NewError("Get expired data failed", err.Error(), watcher)
		return nil, err
	}
	defer resp.Body.Close()
	var datas []*ExpiredData
	_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// watcher.Elastic.NewError("Get expired data failed", err.Error(), nil)
		return nil, err
	}
	err = json.Unmarshal(_bytes, &datas)
	if err != nil {
		// watcher.Elastic.NewError("Get expired data failed", err.Error(), nil)
		return nil, err
	}
	for _, data := range datas {
		data.Datasource = datasource.Code
	}
	return &datas, nil
}

// 从数据库获取数据
func (watcher *WatcherConfig) GetExpiredDataFromSQL(datasource *Datasource) (*[]*ExpiredData, error) {
	if datasource.DB == nil {
		db, err := datasource.Connect()
		if err != nil {
			fmt.Printf("Connect %s db failed: %s\n", datasource.Code, err.Error())
		}
		datasource.DB = db
	}
	err := datasource.DB.Ping()
	if err != nil {
		// watcher.Elastic.NewError("Get expired data failed", err.Error(), map[string]interface{}{
		// 	"DSN": watcher.DataConfig.DSN,
		// })
		fmt.Printf("Ping db failed: %s\n", err.Error())
		return nil, err
	}

	datas := make([]*ExpiredData, 0)
	// 临时存储，获取所有平铺键值对，后续解析
	var temp map[string]interface{}
	rows, err := datasource.DB.Query(watcher.GetExpired)
	if err != nil {
		fmt.Printf("Get %s expited failed: %s\n", watcher.App, err.Error())
	}
	if rows == nil {
		return &datas, nil
	}
	cols, _ := rows.Columns()
	vals := make([]interface{}, len(cols))
	// res := make([]map[string]interface{}, 0)
	for i := range cols {
		vals[i] = new(interface{})
	}
	for rows.Next() {
		rows.Scan(vals...)
		// 解析结果，将平铺键值对转为嵌套对象
		temp = map[string]interface{}{}
		for i, v := range vals {
			col := cols[i]
			switch v := v.(type) {
			case *(interface{}):
				temp[col] = *v
			}
		}
		parsedInterface := parseInterface(temp)
		extend := parsedInterface["Extend"]
		data := ExpiredData{
			Datasource:    datasource.Code,
			WatcherConfig: watcher,
			TimeStamp:     time.Now().Local(),
			Expire1Day:    parseInt(parsedInterface["Expire1Day"]),
			Expire1Week:   parseInt(parsedInterface["Expire1Week"]),
			Expire1Month:  parseInt(parsedInterface["Expire1Month"]),
			Extend:        extend,
		}
		// TEST:
		// data.Expire1Day = rand.Intn(5)
		// data.Expire1Week = rand.Intn(5)
		// data.Expire1Month = rand.Intn(5)
		datas = append(datas, &data)
	}
	return &datas, nil
}

func (watcher *WatcherConfig) GetExpiredDataFunc(datasources *[]*Datasource, elastic *Elastic) func() {
	sources := make([]*Datasource, 0)
	for _, datasource := range *datasources {
		if slices.Contains(watcher.Sources, datasource.Code) {
			sources = append(sources, datasource)
			if len(sources) == len(watcher.Sources) {
				break
			}
		}
	}
	return func() {
		start := time.Now()
		for _, datasource := range sources {
			var getDatas func(datasource *Datasource) (*[]*ExpiredData, error)
			if datasource.Type == DataConfigTypeAPI {
				getDatas = watcher.GetExpiredDataFromAPI
			} else {
				getDatas = watcher.GetExpiredDataFromSQL
			}
			datas, err := getDatas(datasource)
			if err != nil {
				continue
			}
			for _, data := range *datas {
				go elastic.Log(watcher.App, data)
			}
		}
		dur := (time.Now().UnixNano() - start.UnixNano()) / 1e6
		watcher.Mutex.Lock()
		defer watcher.Mutex.Unlock()
		watcher.DurationAvg = (watcher.DurationAvg*int64(watcher.Count) + dur) / (watcher.Count + 1)
		watcher.Count++
		watcher.PrevDuration = dur
	}
}

// 解析为int
func parseInt(i interface{}) int {
	switch v := i.(type) {
	case *interface{}:
		return parseInt(i)
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
func (watcher *WatcherConfig) Enable() error {
	watcher.Mutex.Lock()
	defer watcher.Mutex.Unlock()
	if watcher.Enabled {
		return nil
	}
	watcher.Enabled = true
	// watcher.Start()
	return nil
}

// 启动监控
func (watcher *WatcherConfig) Start(cron *cron.Cron, datasources *[]*Datasource, elastic *Elastic) (cron.EntryID, error) {
	watcher.Mutex.Lock()
	defer watcher.Mutex.Unlock()
	if cron == nil {
		return 0, nil
	}
	if !watcher.Enabled {
		err := errors.New("watcher is disabled")
		// watcher.Elastic.NewError("Start watcher failed", err.Error(), *watcher)
		return 0, err
	}
	if watcher.EntryID != 0 {
		// err := errors.New("watcher is running")
		// watcher.Elastic.NewError("Start watcher failed", err.Error(), *watcher)
		// return err
		return watcher.EntryID, nil
	}
	if watcher.Cron == "" {
		err := errors.New("cron expression is null")
		// watcher.Elastic.NewError("Start watcher failed", err.Error(), *watcher)
		return 0, err
	}
	fun := watcher.GetExpiredDataFunc(datasources, elastic)
	id, err := cron.AddFunc(watcher.Cron, fun)
	if err != nil {
		// watcher.Elastic.NewError("Start watcher failed", err.Error(), *watcher)
		return 0, err
	}
	watcher.EntryID = id
	return id, nil
}

// 禁用监控
func (watcher *WatcherConfig) Disable(cron *cron.Cron) {
	if cron != nil {
		watcher.Stop(cron)
	}
	watcher.Enabled = false
}

// 关闭监控
func (watcher *WatcherConfig) Stop(cron *cron.Cron) {
	watcher.Mutex.Lock()
	defer watcher.Mutex.Unlock()
	if watcher.EntryID != 0 {
		if cron != nil {
			cron.Remove(watcher.EntryID)
		}
		watcher.EntryID = 0
	}
}

type ExpiredData struct {
	Datasource    string         ``                  // 数据源编号
	WatcherConfig *WatcherConfig ``                  // 配置
	TimeStamp     time.Time      `json:"@timestamp"` // 时间戳
	Expire1Day    int            ``                  // 过期1天
	Expire1Week   int            ``                  // 过期7天
	Expire1Month  int            ``                  // 过期1个月
	Extend        interface{}    ``                  // 扩展字段
}
