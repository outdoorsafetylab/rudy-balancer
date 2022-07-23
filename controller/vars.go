package controller

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func stringVar(r *http.Request, name, preset string) string {
	str := mux.Vars(r)[name]
	if str == "" {
		str = r.URL.Query().Get(name)
	}
	if str == "" {
		return preset
	}
	return str
}

func intVar(r *http.Request, name string, preset int) int {
	str := mux.Vars(r)[name]
	if str == "" {
		str = r.URL.Query().Get(name)
	}
	if str == "" {
		return preset
	}
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return preset
	}
	return int(val)
}
