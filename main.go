package main

import (
	"embed"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/max0l/nobody-is-perfect-go/api"
	"github.com/max0l/nobody-is-perfect-go/config"
	"github.com/max0l/nobody-is-perfect-go/frontend"
	"github.com/max0l/nobody-is-perfect-go/game"
	"github.com/max0l/nobody-is-perfect-go/middlewares"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/rs/zerolog/log"
)

//go:embed api/index.html
//go:embed api.yaml
var swaggerUI embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	server := api.NewServerWithGameService(game.NewServiceWithOptions(game.ServiceOptions{
		WordlistPath:       cfg.WordlistPath,
		MaxConcurrentGames: cfg.MaxConcurrentGames,
	}))

	router := gin.New()
	router.Use(ginZerologLogger(), gin.Recovery())

	router.GET("/swagger/*filepath", gin.WrapH(
		http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerUI))),
	))
	frontend.RegisterRoutes(router)

	swagger, err := api.GetSwagger()
	if err != nil {
		log.Fatal().Err(err).Msg("load swagger spec")
	}
	swagger.Servers = openapi3.Servers{{URL: cfg.APIBaseURL}}

	validator := ginmiddleware.OapiRequestValidatorWithOptions(swagger, &ginmiddleware.Options{
		ErrorHandler: validationErrorHandler,
		Options: openapi3filter.Options{
			AuthenticationFunc: middlewares.SessionAuthenticator,
		},
	})
	router.Use(validator)

	sh := api.NewStrictHandler(server, nil)
	api.RegisterHandlersWithOptions(router, sh, api.GinServerOptions{
		Middlewares: []api.MiddlewareFunc{api.MiddlewareFunc(middlewares.SessionAuthMiddleware())},
	})

	s := &http.Server{
		Handler: router,
		Addr:    cfg.Addr(),
	}

	log.Info().Str("addr", cfg.Addr()).Str("api_base_url", cfg.APIBaseURL).Int("max_concurrent_games", cfg.MaxConcurrentGames).Str("wordlist_path", cfg.WordlistPath).Msg("starting http server")
	log.Fatal().Err(s.ListenAndServe()).Msg("http server stopped")
}

func validationErrorHandler(c *gin.Context, message string, statusCode int) {
	if strings.Contains(message, "SecurityRequirementsError") || strings.Contains(message, middlewares.ErrSessionCookieRequired.Error()) {
		statusCode = http.StatusUnauthorized
		message = api.UnauthorizedError
	} else if statusCode == http.StatusBadRequest {
		message = api.BadRequestError
	}

	c.AbortWithStatusJSON(statusCode, gin.H{"error": message})
}

func ginZerologLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.Request.URL.Path
		if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
			path += "?" + rawQuery
		}

		status := c.Writer.Status()
		event := log.Info()
		if status >= http.StatusInternalServerError {
			event = log.Error()
		} else if status >= http.StatusBadRequest {
			event = log.Warn()
		}

		if len(c.Errors) > 0 {
			event.Str("errors", c.Errors.String())
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Int("size", c.Writer.Size()).
			Dur("latency", time.Since(start)).
			Str("client_ip", c.ClientIP()).
			Msg("http request")
	}
}
