package main

import (
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/max0l/nobody-is-perfect-go/api"
	"log"
	"net/http"
)

//go:embed api/index.html
//go:embed api.yaml
var swaggerUI embed.FS

func main() {
	server := api.NewServer()

	router := gin.Default()

	router.GET("/swagger/*filepath", gin.WrapH(
		http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerUI))),
	))

	sh := api.NewStrictHandler(server, nil)
	api.RegisterHandlers(router, sh)

	s := &http.Server{
		Handler: router,
		Addr:    "0.0.0.0:8080",
	}

	log.Fatal(s.ListenAndServe())
}
