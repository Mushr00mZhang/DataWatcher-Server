package controllers

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Action struct {
	Path    string
	Methods []string
	Func    func(w http.ResponseWriter, r *http.Request)
}

func (action Action) BindRouter(router *mux.Router) *mux.Route {
	return router.HandleFunc(action.Path, action.Func).Methods(action.Methods...)
}
