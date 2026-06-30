package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/max0l/nobody-is-perfect-go/api"
	"github.com/max0l/nobody-is-perfect-go/game"
	"github.com/max0l/nobody-is-perfect-go/middlewares"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
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
