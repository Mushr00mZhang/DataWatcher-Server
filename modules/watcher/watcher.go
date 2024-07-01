package watcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"server/modules/es"
	"strconv"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

var ES *es.Config
var Watchers *[]*Config
var Cron *cron.Cron
var (
	SQLTypeSQLServer = "sqlserver"
	SQLTypeMySQL     = "mysql"
	SQLTypeSQLite    = "sqlite"
	SQLTypeOracle    = "oracle"
)

// 监控数据库脚本
type SQLConfig struct {
	Type       string `yaml:"Type"`       // 数据库类型（sqlserver/mysql/sqlite）
	DSN        string `yaml:"DSN"`        // 数据库连接串
	GetExpired string `yaml:"GetExpired"` // 获取过期数据脚本
	GetTasks   string `yaml:"GetTasks"`   // 获取任务列表脚本
	ResetTasks string `yaml:"ResetTasks"` // 重置任务状态脚本
}

// 监控配置
type Config struct {
	Module     string       `yaml:"Module"`           // 模块
	System     string       `yaml:"System"`           // 系统
	Provider   string       `yaml:"Provider"`         // 提供方
	Requester  string       `yaml:"Requester"`        // 请求方
	Type       string       `yaml:"Type"`             // 类型（Push/Pull）
	Method     string       `yaml:"Method"`           // 承载方式
	App        string       `yaml:"App"`              // 应用名称
	Desc       string       `yaml:"Desc"`             // 描述
	Interface  string       `yaml:"Interface"`        // 接口名称
	ConfigPath string       `yaml:"ConfigPath"`       // 配置路径
	Tags       []string     `yaml:"Tags"`             // 标签
	SQL        SQLConfig    `yaml:"SQL"`              // 数据库脚本
	Extend     interface{}  `yaml:"Extend"`           // 扩展字段
	Cron       string       `yaml:"Cron"`             // Cron表达式
	Enabled    bool         `yaml:"Enabled" `         // 是否启用
	DB         *gorm.DB     `yaml:"-" json:"-"`       // 连接池
	EntryID    cron.EntryID `yaml:"-" json:"EntryID"` // Cron运行时ID
}

// 数据库连接
func (conf *Config) Connect() error {
	if conf.DB != nil {
		return nil
	}
	if conf.SQL.DSN == "" {
		ES.NewError("DSN not found", "", nil)
		return errors.New("DSN not found")
	}
	if conf.SQL.Type == "" {
		conf.SQL.Type = SQLTypeSQLServer
	}
	var open func(dsn string) gorm.Dialector
	switch conf.SQL.Type {
	case SQLTypeSQLServer:
		open = sqlserver.Open
	case SQLTypeMySQL:
		open = mysql.Open
	case SQLTypeSQLite:
		open = sqlite.Open
		// case SQLTypeOracle:
		// 	open = oracle.Open
	}
	if open != nil {
		db, err := gorm.Open(open(conf.SQL.DSN), &gorm.Config{})
		if err != nil {
			ES.NewError(fmt.Sprintf("Open %s failed", conf.SQL.Type), err.Error(), map[string]interface{}{
				"Type": conf.SQL.Type,
				"DSN":  conf.SQL.DSN,
			})
			return err
		}
		conf.DB = db
	}
	return nil
}

// 获取数据
func (conf *Config) GetExpiredData() (*Data, error) {
	err := conf.Connect()
	if err != nil {
		ES.NewError("Connect to db error", err.Error(), map[string]interface{}{
			"DSN": conf.SQL.DSN,
		})
		return nil, err
	}
	// 临时存储，获取所有平铺键值对，后续解析
	temp := map[string]interface{}{}
	conf.DB.Raw(conf.SQL.GetExpired).First(&temp)
	// 解析结果，将平铺键值对转为嵌套对象
	parsedInterface := parseInterface(temp)
	extend := parsedInterface["Extend"]
	data := Data{
		Config:       conf,
		TimeStamp:    time.Now().Local(),
		Expire1Day:   parseInt(parsedInterface["Expire1Day"]),
		Expire1Week:  parseInt(parsedInterface["Expire1Week"]),
		Expire1Month: parseInt(parsedInterface["Expire1Month"]),
		Extend:       extend,
	}
	// TEST:
	// data.Expire1Day = rand.Intn(5)
	// data.Expire1Week = data.Expire1Week + rand.Intn(5)
	// data.Expire1Month = data.Expire1Week + rand.Intn(5)

	return &data, nil
}

func (conf *Config) LogExpiredData() {
	data, err := conf.GetExpiredData()
	if err != nil {
		return
	}
	ES.Log(conf.App, data)
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
func (conf *Config) Enable() error {
	if conf.Enabled && conf.EntryID != 0 {
		return nil
	}
	if conf.Cron == "" {
		err := errors.New("cron expression is null")
		ES.NewError("Enable watcher failed", err.Error(), *conf)
		return err
	}
	id, err := Cron.AddFunc(conf.Cron, conf.LogExpiredData)
	if err != nil {
		ES.NewError("Enable watcher failed", err.Error(), *conf)
		return err
	}
	conf.EntryID = id
	conf.Enabled = true
	return nil
}

// 禁用监控
func (conf *Config) Disable() {
	conf.DB = nil
	Cron.Remove(conf.EntryID)
	conf.EntryID = 0
	conf.Enabled = false
}

type Data struct {
	Config       *Config     ``                 // 配置
	TimeStamp    time.Time   `json:"timestamp"` // 时间戳
	Expire1Day   int         ``                 // 过期1天
	Expire1Week  int         ``                 // 过期7天
	Expire1Month int         ``                 // 过期1个月
	Extend       interface{} ``                 // 扩展字段
}

// 上传到ES
func (data *Data) ToES() error {
	return ES.Log(data.Config.App, data)
}

// 绑定Router
func BindRouter(base *mux.Router) {
	r := base.PathPrefix("/watchers").Subrouter()
	r.HandleFunc("", GetWatchers).Methods(http.MethodGet)
	r.HandleFunc("/{app}", GetWatcher).Methods(http.MethodGet)
	r.HandleFunc("/{app}", CreateWatcher).Methods(http.MethodPost)
	r.HandleFunc("/{app}", UpdateWatcher).Methods(http.MethodPut)
	r.HandleFunc("/{app}", DeleteWatcher).Methods(http.MethodDelete)
	r.HandleFunc("/{app}/enable", EnableWatcher).Methods(http.MethodPatch)
	r.HandleFunc("/{app}/disable", DisableWatcher).Methods(http.MethodPatch)
}

// 获取监控列表
func GetWatchers(w http.ResponseWriter, r *http.Request) {
	_bytes, err := json.Marshal(Watchers)
	if err != nil {
		ES.NewError("Get watchers failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write(_bytes)
	w.WriteHeader(200)
}

// 获取监控
func GetWatcher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]
	for _, watcher := range *Watchers {
		if watcher.App == app {
			_bytes, err := json.Marshal(watcher)
			if err != nil {
				ES.NewError("Get watcher failed", err.Error(), map[string]interface{}{
					"App": app,
				})
			}
			w.Header().Add("Content-Type", "application/json;charset=UTF-8")
			w.Write(_bytes)
			w.WriteHeader(200)
			return
		}
	}
	w.WriteHeader(404)
}

// 创建监控
func CreateWatcher(w http.ResponseWriter, r *http.Request) {
	var conf Config
	_bytes, err := io.ReadAll(r.Body)
	if err != nil {
		ES.NewError("Create watcher failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	err = json.Unmarshal(_bytes, &conf)
	if err != nil {
		ES.NewError("Create watcher failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	watchers := append(*Watchers, &conf)
	Watchers = &watchers
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write([]byte(conf.App))
	w.WriteHeader(200)
}

// 更新监控
func UpdateWatcher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]
	var conf Config
	_bytes, err := io.ReadAll(r.Body)
	if err != nil {
		ES.NewError("Update watcher failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	err = json.Unmarshal(_bytes, &conf)
	if err != nil {
		ES.NewError("Update watcher failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	watchers := *Watchers
	for i, watcher := range *Watchers {
		if watcher.App == app {
			if (watcher.SQL.Type != conf.SQL.Type || watcher.SQL.DSN != conf.SQL.DSN) && watcher.DB != nil {
				watcher.DB = nil
			}
			watchers[i] = &conf
			Watchers = &watchers
			w.Header().Add("Content-Type", "application/json;charset=UTF-8")
			w.Write([]byte(watcher.App))
			w.WriteHeader(200)
			return
		}
	}
	w.WriteHeader(404)
}

// 删除监控
func DeleteWatcher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]
	i := 0
	watchers := *Watchers
	for _, watcher := range *Watchers {
		if watcher.App != app {
			watchers[i] = watcher
			i++
		} else if watcher.DB != nil {
			watcher.DB = nil
			Cron.Remove(watcher.EntryID)
		}
	}
	watchers = watchers[:i]
	Watchers = &watchers
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write([]byte(app))
	w.WriteHeader(200)
}

// 启用监控
func EnableWatcher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]
	for _, watcher := range *Watchers {
		if watcher.App == app {
			err := watcher.Enable()
			if err != nil {
				w.Header().Add("Content-Type", "application/json;charset=UTF-8")
				w.Write([]byte(err.Error()))
				w.WriteHeader(500)
				return
			}
			w.Header().Add("Content-Type", "application/json;charset=UTF-8")
			w.Write([]byte(watcher.App))
			w.WriteHeader(200)
			return
		}
	}
	w.WriteHeader(404)
}

// 禁用监控
func DisableWatcher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]
	for _, watcher := range *Watchers {
		if watcher.App == app {
			watcher.Disable()
			w.Header().Add("Content-Type", "application/json;charset=UTF-8")
			w.Write([]byte(watcher.App))
			w.WriteHeader(200)
			return
		}
	}
	w.WriteHeader(404)
}
