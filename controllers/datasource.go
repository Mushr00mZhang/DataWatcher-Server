package controllers

import (
	"encoding/json"
	"net/http"
	"server/services"

	"github.com/gorilla/mux"
)

type DatasourceController struct {
	DatasourceService *services.DatasourceService
}

func NewDatasourceController(datasourceService *services.DatasourceService) *DatasourceController {
	return &DatasourceController{
		DatasourceService: datasourceService,
	}
}

// 绑定Router
func (controller DatasourceController) BindRouter(base *mux.Router) {
	subrouter := base.PathPrefix("/datasources").Subrouter()
	subrouter.HandleFunc("", controller.GetDatasources).Methods(http.MethodGet)
}

// 获取数据源列表
func (controller DatasourceController) GetDatasources(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")
	datasources := controller.DatasourceService.GetDatasources()
	bytes, err := json.Marshal(datasources)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	w.Write(bytes)
	w.WriteHeader(200)
}
