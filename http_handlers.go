package main

// this file contains implementation of HTTP handlers - REST API

import (
	"github.com/labstack/echo"
	"net/http"
)

var service Service

func NewHTTPRouter(_service Service) *echo.Echo {
	service = _service

	router := echo.New()
	router.GET("/health", healthCheckHandler)

	return router
}

func healthCheckHandler(c echo.Context) error {
	message := c.QueryParam("message")
	service.Test(message)
	return c.String(http.StatusOK, "I am up and running!")
}
