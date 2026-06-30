package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/max0l/nobody-is-perfect-go/api"
	"github.com/max0l/nobody-is-perfect-go/config"
	"github.com/max0l/nobody-is-perfect-go/game"
	"github.com/max0l/nobody-is-perfect-go/middlewares"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/rs/zerolog"
)

func TestOpenAPIValidatorUsesRelativeServerURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	swagger, err := api.GetSwagger()
	if err != nil {
		t.Fatalf("GetSwagger returned error: %v", err)
	}
	swagger.Servers = openapi3.Servers{{URL: "/"}}

	router := gin.New()
	router.Use(ginmiddleware.OapiRequestValidatorWithOptions(swagger, &ginmiddleware.Options{
		ErrorHandler:          validationErrorHandler,
		SilenceServersWarning: true,
		Options: openapi3filter.Options{
			AuthenticationFunc: middlewares.SessionAuthenticator,
		},
	}))
	api.RegisterHandlersWithOptions(router, api.NewStrictHandler(api.NewServerWithGameService(game.NewService()), nil), api.GinServerOptions{
		Middlewares: []api.MiddlewareFunc{api.MiddlewareFunc(middlewares.SessionAuthMiddleware())},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/create/user", strings.NewReader(`{"username":"Max"}`))
	request.Host = "example.com"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusCreated, response.Code, response.Body.String())
	}
}

func TestZerologLevelMapsConfiguredLevel(t *testing.T) {
	tests := map[string]zerolog.Level{
		config.LogLevelTrace:    zerolog.TraceLevel,
		config.LogLevelDebug:    zerolog.DebugLevel,
		config.LogLevelInfo:     zerolog.InfoLevel,
		config.LogLevelWarn:     zerolog.WarnLevel,
		config.LogLevelError:    zerolog.ErrorLevel,
		config.LogLevelDisabled: zerolog.Disabled,
		"":                      zerolog.InfoLevel,
	}

	for level, expected := range tests {
		if actual := zerologLevel(level); actual != expected {
			t.Fatalf("expected %q to map to %s, got %s", level, expected, actual)
		}
	}
}

func TestPrintVersionGreetingIncludesTrimmedVersion(t *testing.T) {
	var buf bytes.Buffer

	if err := printVersionGreeting(&buf, "1.2.3\n"); err != nil {
		t.Fatalf("printVersionGreeting returned error: %v", err)
	}

	expected := "Hello from nobody-is-perfect-go 1.2.3\n"
	if buf.String() != expected {
		t.Fatalf("expected greeting %q, got %q", expected, buf.String())
	}
}
