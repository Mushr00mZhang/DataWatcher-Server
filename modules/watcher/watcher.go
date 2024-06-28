package watcher

import (
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"server/modules/es"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

var ES *es.Config
var Watchers *[]*Config

type DateFilter struct {
	SDate string
	EDate string
}

func GetDateFilter(prev int) DateFilter {
	now := time.Now().Local()
	return DateFilter{
		SDate: now.AddDate(0, 0, -prev).Format("2006-01-02"),
		EDate: now.Format("2006-01-02"),
	}
}

type Config struct {
	Module      string                 `yaml:"Module"`      // 模块
	System      string                 `yaml:"System"`      // 系统
	Provider    string                 `yaml:"Provider"`    // 提供方
	Requester   string                 `yaml:"Requester"`   // 请求方
	Code        string                 `yaml:"Code"`        // 编号
	Desc        string                 `yaml:"Desc"`        // 描述
	Method      string                 `yaml:"Method"`      // 承载方式
	App         string                 `yaml:"App"`         // 应用名称
	Tags        []string               `yaml:"Tags"`        // 标签
	DSN         string                 `yaml:"DSN"`         // 数据库连接串
	SQLTemplate string                 `yaml:"SQLTemplate"` // 取数SQL模板
	Extend      map[string]interface{} `yaml:"Extend"`      // 扩展字段
	Enabled     bool                   `yaml:"Enabled" `    // 是否启用
	DB          *gorm.DB               `yaml:"-" json:"-"`  // 连接池
}

func (conf *Config) Connect() error {
	if conf.DSN == "" {
		ES.NewError("DSN not found", "", nil)
		return errors.New("DSN not found")
	}
	db, err := gorm.Open(sqlserver.Open(conf.DSN), &gorm.Config{})
	if err != nil {
		ES.NewError("Open sqlserver failed", err.Error(), map[string]interface{}{
			"DSN": conf.DSN,
		})
		return err
	}
	conf.DB = db
	return nil
}

func (conf *Config) GetData() (*Data, error) {
	err := conf.Connect()
	if err != nil {
		ES.NewError("Connect to db error", err.Error(), map[string]interface{}{
			"DSN": conf.DSN,
		})
		return nil, err
	}
	data := Data{
		Config:    conf,
		TimeStamp: time.Now().Local(),
	}
	conf.DB.Raw(conf.SQLTemplate, GetDateFilter(1)).Scan(&data.Over1Day)
	conf.DB.Raw(conf.SQLTemplate, GetDateFilter(3)).Scan(&data.Over3Day)
	conf.DB.Raw(conf.SQLTemplate, GetDateFilter(7)).Scan(&data.Over7Day)

	data.Over1Day = rand.Intn(5)
	data.Over3Day = data.Over3Day + rand.Intn(5)
	data.Over7Day = data.Over3Day + rand.Intn(5)
	return &data, nil
}

type Data struct {
	Config    *Config
	TimeStamp time.Time `json:"timestamp"`
	Over1Day  int
	Over3Day  int
	Over7Day  int
}

func (data *Data) ToES() error {
	return ES.Log(data.Config.App, data)
}

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
func DeleteWatcher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]
	i := 0
	watchers := *Watchers
	for _, watcher := range *Watchers {
		if watcher.App != app {
			watchers[i] = watcher
			i++
		}
	}
	watchers = watchers[:i]
	Watchers = &watchers
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write([]byte(app))
	w.WriteHeader(200)
}
func EnableWatcher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]
	for _, watcher := range *Watchers {
		if watcher.App == app {
			watcher.Enabled = true
			w.Header().Add("Content-Type", "application/json;charset=UTF-8")
			w.Write([]byte(watcher.App))
			w.WriteHeader(200)
			return
		}
	}
	w.WriteHeader(404)
}
func DisableWatcher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]
	for _, watcher := range *Watchers {
		if watcher.App == app {
			watcher.Enabled = false
			w.Header().Add("Content-Type", "application/json;charset=UTF-8")
			w.Write([]byte(watcher.App))
			w.WriteHeader(200)
			return
		}
	}
	w.WriteHeader(404)
}
