package main

// this file contains implementation of HTTP handlers - REST API

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

var service Service

func NewHTTPRouter(_service Service) *mux.Router {
	service = _service

	router := mux.NewRouter()

	router.HandleFunc("/health", healthCheckHandler).
		Methods("GET")

	return router
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Query().Get("message")
	service.Test(message)

	fmt.Fprintf(w, "I am up and running")
}
