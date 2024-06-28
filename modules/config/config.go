package config

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"server/modules/es"
	"server/modules/watcher"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

const ConfigPath = "./config.yml"

var Conf *Config

type Config struct {
	Watchers *[]*watcher.Config `yaml:"Watchers"` // 监控列表
	ES       *es.Config         `yaml:"ES"`       // Elasticsearch
	Interval int                `yaml:"Interval"` // 轮询时间（分钟）
	Status   int8
}

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
func Save() {
	bytes, err := yaml.Marshal(Conf)
	if err != nil {
		log.Fatalf("Save config file failed: %v", err)
	}
	os.WriteFile(ConfigPath, bytes, 0666)
}
func (conf *Config) Init() {
	conf.ES.Init()
	watcher.ES = conf.ES
	watcher.Watchers = conf.Watchers
}
func (conf *Config) Run(ch chan struct{}) {
	if conf.Status == 1 {
		Conf.LogDatas()
		dur, _ := time.ParseDuration(fmt.Sprintf("%vm", Conf.Interval))
		time.Sleep(dur)
		conf.Run(ch)
	} else {
		close(ch)
	}
}
func (conf *Config) Stop() {
	conf.Status = 0
}
func (conf *Config) LogDatas() {
	for _, watcher := range *conf.Watchers {
		err := watcher.Connect()
		if err != nil {
			continue
		}
		data, err := watcher.GetData()
		if err != nil {
			continue
		}
		conf.ES.Log(watcher.App, data)
	}
}

func BindRouter(base *mux.Router) {
	r := base.PathPrefix("/config").Subrouter()
	r.HandleFunc("", GetConfig).Methods(http.MethodGet)
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
