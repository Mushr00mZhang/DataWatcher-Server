package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server/modules"
	"server/services"
	"strings"

	"github.com/gorilla/mux"
)

type WatcherController struct {
	WatcherService *services.WatcherService
}

func NewWatcherController(watcherService *services.WatcherService) *WatcherController {
	return &WatcherController{
		WatcherService: watcherService,
	}
}

// 绑定Router
func (controller WatcherController) BindRouter(base *mux.Router) {
	subrouter := base.PathPrefix("/watchers").Subrouter()
	subrouter.HandleFunc("", controller.GetWatchers).Methods(http.MethodGet)
	subrouter.HandleFunc("/entries", controller.GetEntries).Methods(http.MethodGet)
	subrouter.HandleFunc("/{app}/entry", controller.GetWatcherEntry).Methods(http.MethodGet)
	subrouter.HandleFunc("/{app}", controller.GetWatcher).Methods(http.MethodGet)
	subrouter.HandleFunc("/{app}", controller.CreateWatcher).Methods(http.MethodPost)
	subrouter.HandleFunc("/{app}", controller.UpdateWatcher).Methods(http.MethodPut)
	subrouter.HandleFunc("/{app}", controller.DeleteWatcher).Methods(http.MethodDelete)
	subrouter.HandleFunc("/{app}/enable", controller.EnableWatcher).Methods(http.MethodPatch)
	subrouter.HandleFunc("/{app}/disable", controller.DisableWatcher).Methods(http.MethodPatch)
	subrouter.HandleFunc("/{app}/start", controller.StartWatcher).Methods(http.MethodPatch)
	subrouter.HandleFunc("/{app}/stop", controller.StopWatcher).Methods(http.MethodPatch)
}

// 获取监控列表
func (controller WatcherController) GetWatchers(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	watchers := controller.WatcherService.GetWatchers()
	bytes, err := json.Marshal(watchers)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	w.Write(bytes)
	w.WriteHeader(200)
}

// 获取监控
func (controller WatcherController) GetWatcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	app := vars["app"]
	watcher, err := controller.WatcherService.GetWatcher(app)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	bytes, _ := json.Marshal(watcher)
	// if err != nil {
	// 	ES.NewError("Get watcher failed", err.Error(), map[string]interface{}{
	// 		"App": app,
	// 	})
	// }
	w.Write(bytes)
	w.WriteHeader(200)
}

// 创建监控
func (controller WatcherController) CreateWatcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	var new modules.WatcherConfig
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		// ES.NewError("Create watcher failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	err = json.Unmarshal(bytes, &new)
	if err != nil {
		// ES.NewError("Create watcher failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	err = controller.WatcherService.CreateWatcher(&new)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(new.App))
	w.WriteHeader(201)
}

// 更新监控
func (controller WatcherController) UpdateWatcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	app := vars["app"]
	var new modules.WatcherConfig
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		// ES.NewError("Update watcher failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	err = json.Unmarshal(bytes, &new)
	if err != nil {
		// ES.NewError("Update watcher failed", err.Error(), nil)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	err = controller.WatcherService.UpdateWatcher(app, &new)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(new.App))
	w.WriteHeader(200)
}

// 删除监控
func (controller WatcherController) DeleteWatcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	app := vars["app"]
	err := controller.WatcherService.DeleteWatcher(app)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(app))
	w.WriteHeader(200)
}

// 启用监控
func (controller WatcherController) EnableWatcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	app := vars["app"]
	err := controller.WatcherService.EnableWatcher(app)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(app))
	w.WriteHeader(200)
}

// 禁用监控
func (controller WatcherController) DisableWatcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	app := vars["app"]
	err := controller.WatcherService.DisableWatcher(app)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(app))
	w.WriteHeader(200)
}

// 开始监控
func (controller WatcherController) StartWatcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	app := vars["app"]
	id, err := controller.WatcherService.StartWatcher(app)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(fmt.Sprintf("%d", id)))
	w.WriteHeader(200)
}

// 停止监控
func (controller WatcherController) StopWatcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	app := vars["app"]
	err := controller.WatcherService.StopWatcher(app)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	w.Write([]byte(app))
	w.WriteHeader(200)
}

// 获取监控列表状态
func (controller WatcherController) GetEntries(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	query := r.URL.Query()
	apps := strings.Split(query.Get("apps"), ",")
	entries, err := controller.WatcherService.GetEntries(apps)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	bytes, err := json.Marshal(entries)
	if err != nil {
		// ES.NewError("Get watcher entry failed", err.Error(), res)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	w.Write(bytes)
	w.WriteHeader(200)
}

// 获取监控状态
func (controller WatcherController) GetWatcherEntry(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	vars := mux.Vars(r)
	app := vars["app"]
	entry, err := controller.WatcherService.GetWatcherEntry(app)
	if err != nil {
		w.Write([]byte(err.Error()))
		if err == modules.ErrWatcherNotFound {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(500)
		return
	}
	bytes, err := json.Marshal(entry)
	if err != nil {
		// ES.NewError("Get watcher entry failed", err.Error(), res)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	w.Write(bytes)
	w.WriteHeader(200)
}
