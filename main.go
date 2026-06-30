package main

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed api/index.html
//go:embed api.yaml
var swaggerUI embed.FS

//go:embed VERSION
var version string

const shutdownTimeout = 10 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	configureLogging(cfg.LogFormat, cfg.LogLevel)
	gin.SetMode(cfg.GinMode)
	server := api.NewServerWithGameService(game.NewServiceWithOptions(game.ServiceOptions{
		WordlistPath:       cfg.WordlistPath,
		MaxConcurrentGames: cfg.MaxConcurrentGames,
	}))

	router := gin.New()
	router.Use(ginZerologLogger(), gin.Recovery(), middlewares.SecurityHeadersMiddleware())

	router.GET("/swagger/*filepath", gin.WrapH(
		http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerUI))),
	))
	frontend.RegisterRoutes(router)

	swagger, err := api.GetSwagger()
	if err != nil {
		log.Fatal().Err(err).Msg("load swagger spec")
	}
	swagger.Servers = openapi3.Servers{{URL: "/"}}

	validator := ginmiddleware.OapiRequestValidatorWithOptions(swagger, &ginmiddleware.Options{
		ErrorHandler:          validationErrorHandler,
		SilenceServersWarning: true,
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

	log.Info().
		Str("version", strings.TrimSpace(version)).
		Str("listen_addr", cfg.Addr()).
		Int("max_concurrent_games", cfg.MaxConcurrentGames).
		Str("wordlist_path", cfg.WordlistPath).
		Str("log_format", cfg.LogFormat).
		Str("log_level", cfg.LogLevel).
		Str("gin_mode", cfg.GinMode).
		Msg("starting http server")
	if err := runHTTPServer(s); err != nil {
		log.Fatal().Err(err).Msg("http server stopped")
	}
}

func runHTTPServer(s *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	stop()
	log.Info().Dur("timeout", shutdownTimeout).Msg("shutting down http server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return <-serverErr
}

func configureLogging(format, level string) {
	switch format {
	case config.LogFormatText:
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})
	case config.LogFormatTextColor:
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: false})
	default:
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(zerologLevel(level))
}

func zerologLevel(level string) zerolog.Level {
	switch level {
	case config.LogLevelTrace:
		return zerolog.TraceLevel
	case config.LogLevelDebug:
		return zerolog.DebugLevel
	case config.LogLevelWarn:
		return zerolog.WarnLevel
	case config.LogLevelError:
		return zerolog.ErrorLevel
	case config.LogLevelDisabled:
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
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
