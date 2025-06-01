package main

import (
	"github.com/gin-gonic/gin"
	"github.com/max0l/nobody-is-perfect-go/api"
	"log"
	"net/http"
)

func main() {
	server := api.NewServer()

	router := gin.Default()

	sh := api.NewStrictHandler(server, nil)
	api.RegisterHandlers(router, sh)

	s := &http.Server{
		Handler: router,
		Addr:    "0.0.0.0:8080",
	}

	log.Fatal(s.ListenAndServe())
}
