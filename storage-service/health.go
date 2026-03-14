package main

import "net/http"

type HealthCheck struct {
}

func (h HealthCheck) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func NewHealthCheck() *HealthCheck {
	return &HealthCheck{}
}
