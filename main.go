package main

import (
	"fmt"
	"net/http"
	"server/modules/config"
	"server/modules/watcher"

	"github.com/gorilla/mux"
)

func main() {
	config.Read()
	config.Conf.Init()
	fmt.Printf("--- conf:\n%v\n%v\n", *config.Conf.Watchers, *config.Conf.ES)
	// config.Conf.Start()
	// c := cron.New(
	// 	cron.WithParser(
	// 		cron.NewParser(cron.Minute | cron.Hour),
	// 	),
	// )
	// c.AddFunc("0/1 *", func() { fmt.Printf("%v\n", time.Now()) })
	// c.Start()
	// select {}
	// var wg sync.WaitGroup
	// wg.Add(1)
	// done := make(chan struct{})

	// go func() {
	// 	defer wg.Done() // 在goroutine结束时调用，表示轮询已完成
	// 	config.Conf.Status = config.ConfigStatusRunning
	// 	config.Conf.Run(done)
	// }()

	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()
	config.BindRouter(apiRouter)
	watcher.BindRouter(apiRouter)
	http.ListenAndServe(":8080", router)
}
