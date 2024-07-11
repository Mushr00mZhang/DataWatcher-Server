package main

import (
	"log"
	"net/http"
	"os"
	"server/controllers"
	"server/modules"
	"server/services"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	_ "time/tzdata"
)

func main() {
	// 设置timezone，默认Asia/Shanghai
	tz := os.Getenv("TZ")
	if strings.TrimSpace(tz) == "" {
		tz = "Asia/Shanghai"
	}
	time.LoadLocation(tz)

	conf := ReadConfig()
	scheduler := &modules.Scheduler{
		Status: modules.SchedulerStatusStop,
	}
	scheduler.Init()
	elastic := conf.Elastic
	elastic.Init()
	// elasticService := services.NewElasticService(elastic)
	schedulerService := services.NewSchedulerService(conf.Watchers, conf.Datasources, scheduler, elastic)
	watcherService := services.NewWatcherService(conf.Watchers, conf.Datasources, scheduler, elastic)

	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()
	watcherController := controllers.NewWatcherController(watcherService)
	schedulerController := controllers.NewSchedulerController(schedulerService)
	watcherController.BindRouter(apiRouter)
	schedulerController.BindRouter(apiRouter)
	http.ListenAndServe(":8080", router)
}

// 读取配置文件
func ReadConfig() *modules.Config {
	bytes, err := os.ReadFile(modules.ConfigPath)
	if err != nil {
		log.Fatalf("Read config file failed: %v", err)
		panic("Config file not found.")
	}
	var conf modules.Config
	err = yaml.Unmarshal(bytes, &conf)
	if err != nil {
		log.Fatalf("Parse config file failed: %v", err)
		panic("Config file cannot parse.")
	}
	return &conf
}

// 保存配置文件
func SaveConfig(conf *modules.Config) {
	bytes, err := yaml.Marshal(conf)
	if err != nil {
		log.Fatalf("Save config file failed: %v", err)
	}
	os.WriteFile(modules.ConfigPath, bytes, 0666)
}
