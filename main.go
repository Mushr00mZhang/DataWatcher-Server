package main

import (
	"net/http"
	"os"
	"server/controllers"
	"server/modules"
	"server/services"
	"strings"
	"time"

	"github.com/gorilla/mux"

	_ "time/tzdata"
)

func main() {
	// 设置timezone，默认Asia/Shanghai
	tz := os.Getenv("TZ")
	if strings.TrimSpace(tz) == "" {
		tz = "Asia/Shanghai"
	}
	time.LoadLocation(tz)

	conf := modules.NewConfig()
	scheduler := &modules.Scheduler{
		Status: modules.SchedulerStatusStop,
	}
	scheduler.Init()
	elastic := conf.Elastic
	elastic.Init()
	// elasticService := services.NewElasticService(elastic)
	datasourceService := services.NewDatasourceService(conf.Datasources)
	schedulerService := services.NewSchedulerService(conf.Watchers, conf.Datasources, scheduler, elastic)
	watcherService := services.NewWatcherService(conf, conf.Watchers, conf.Datasources, scheduler, elastic)
	go func() {
		schedulerService.Start()
	}()
	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()
	datasourceController := controllers.NewDatasourceController(datasourceService)
	watcherController := controllers.NewWatcherController(watcherService)
	schedulerController := controllers.NewSchedulerController(schedulerService)
	datasourceController.BindRouter(apiRouter)
	watcherController.BindRouter(apiRouter)
	schedulerController.BindRouter(apiRouter)
	http.ListenAndServe(":8080", router)
}
