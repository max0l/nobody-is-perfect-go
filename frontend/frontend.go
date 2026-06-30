package frontend

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

//go:embed templates/*.html static/*
var files embed.FS

var indexTemplate = template.Must(template.ParseFS(files, "templates/index.html"))

type indexData struct {
	GameID string
}

func RegisterRoutes(router gin.IRouter) {
	router.GET("/", renderIndex(""))
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/game/:gameId", func(c *gin.Context) {
		renderIndex(c.Param("gameId"))(c)
	})
	router.GET("/assets/*filepath", gin.WrapH(http.StripPrefix("/assets/", http.FileServer(http.FS(mustSub(files, "static"))))))
}

func renderIndex(gameID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := indexTemplate.Execute(c.Writer, indexData{GameID: gameID}); err != nil {
			log.Error().Err(err).Msg("render frontend")
			c.Status(http.StatusInternalServerError)
		}
	}
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
