package controllers

import (
	"net/http"
	"server/services"

	"github.com/gorilla/mux"
)

type SchedulerController struct {
	SchedulerService *services.SchedulerService
}

func NewSchedulerController(schedulerService *services.SchedulerService) *SchedulerController {
	return &SchedulerController{
		SchedulerService: schedulerService,
	}
}

// 绑定Router
func (controller SchedulerController) BindRouter(base *mux.Router) {
	subrouter := base.PathPrefix("/scheduler").Subrouter()
	subrouter.HandleFunc("/start", controller.Start).Methods(http.MethodPatch)
	subrouter.HandleFunc("/stop", controller.Stop).Methods(http.MethodPatch)
}

// 开启调度
func (controller SchedulerController) Start(w http.ResponseWriter, r *http.Request) {
	// goroutine执行
	go controller.SchedulerService.Start()
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write([]byte("true"))
	w.WriteHeader(200)
}

// 停止调度
func (controller SchedulerController) Stop(w http.ResponseWriter, r *http.Request) {
	controller.SchedulerService.Stop()
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	w.Write([]byte("true"))
	w.WriteHeader(200)
}
