package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func setupRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", serveImages).Methods("GET")
	return r
}

func serveImages(w http.ResponseWriter, r *http.Request) {
	// Вызов функции контроллера для обработки запроса
	handleWatermarkedImages(w, r)
}
