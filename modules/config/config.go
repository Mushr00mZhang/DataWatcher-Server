package config

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"server/modules/es"
	"server/modules/watcher"

	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

const ConfigPath = "./config.yml"

var Conf *Config
var (
	ConfigStatusStop    int8 = 0
	ConfigStatusRunning int8 = 1
)

type Config struct {
	Watchers *[]*watcher.Config `yaml:"Watchers"` // 监控列表
	ES       *es.Config         `yaml:"ES"`       // Elasticsearch
	Cron     *cron.Cron
	Status   int8
}

// 读取配置文件
func Read() {
	bytes, err := os.ReadFile(ConfigPath)
	if err != nil {
		log.Fatalf("Read config file failed: %v", err)
		panic("Config file not found.")
	}
	conf := Config{}
	err = yaml.Unmarshal(bytes, &conf)
	if err != nil {
		log.Fatalf("Parse config file failed: %v", err)
		panic("Config file cannot parse.")
	}
	Conf = &conf
}

// 保存配置文件
func Save() {
	bytes, err := yaml.Marshal(Conf)
	if err != nil {
		log.Fatalf("Save config file failed: %v", err)
	}
	os.WriteFile(ConfigPath, bytes, 0666)
}

// 初始化
func (conf *Config) Init() {
	conf.ES.Init()
	watcher.ES = conf.ES
	watcher.Watchers = conf.Watchers
}

// 开始调度
func (conf *Config) Start() {
	if conf.Status == ConfigStatusStop {
		fmt.Printf("GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
		conf.Status = ConfigStatusRunning
		if conf.Cron == nil {
			conf.Cron = cron.New(
				cron.WithParser(
					cron.NewParser(cron.Minute | cron.Hour),
				),
			)
			watcher.Cron = conf.Cron
		}
		for _, watcher := range *Conf.Watchers {
			if watcher.EntryID != 0 {
				// watcher.Stop()
				continue
			}
			err := watcher.Start()
			if err != nil {
				continue
			}
		}
		conf.Cron.Start()
	}
}

//	func (conf *Config) Run(ch chan struct{}) {
//		if conf.Status == ConfigStatusStop {
//			fmt.Printf("GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
//			conf.Status = ConfigStatusRunning
//			if conf.Cron == nil {
//				conf.Cron = cron.New()
//			}
//			for _, watcher := range *Conf.Watchers {
//				if watcher.EntryID != 0 {
//					continue
//				}
//				err := watcher.Enable()
//				if err != nil {
//					continue
//				}
//			}
//			conf.Cron.Start()
//		} else {
//			conf.Stop()
//			close(ch)
//		}
//	}
func (conf *Config) Stop() {
	ctx := conf.Cron.Stop()
	ctx.Done()
	for _, watcher := range *conf.Watchers {
		watcher.EntryID = 0
	}
	conf.Status = ConfigStatusStop
}

func BindRouter(base *mux.Router) {
	r := base.PathPrefix("/config").Subrouter()
	r.HandleFunc("", GetConfig).Methods(http.MethodGet)
	r.HandleFunc("/start", Start).Methods(http.MethodPatch)
	r.HandleFunc("/stop", Stop).Methods(http.MethodPatch)
}
func GetConfig(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.Marshal(Conf)
	if err != nil {
		Conf.ES.NewError("Get config failed", err.Error(), nil)
	}
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write(bytes)
	w.WriteHeader(200)
}

// 启动调度
func Start(w http.ResponseWriter, r *http.Request) {
	go Conf.Start()
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write([]byte("true"))
	w.WriteHeader(200)
}

// 停止调度
func Stop(w http.ResponseWriter, r *http.Request) {
	Conf.Stop()
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write([]byte("true"))
	w.WriteHeader(200)
}
