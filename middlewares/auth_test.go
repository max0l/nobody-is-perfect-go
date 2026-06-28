package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/max0l/nobody-is-perfect-go/api"
)

func TestSessionAuthMiddlewareAllowsUnprotectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SessionAuthMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(response, req)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestSessionAuthMiddlewareRejectsProtectedRoutesWithoutCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(api.SessionCookieScopes, []string{})
		c.Next()
	})
	router.Use(SessionAuthMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(response, req)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestSessionAuthMiddlewareStoresSessionCookieValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(api.SessionCookieScopes, []string{})
		c.Next()
	})
	router.Use(SessionAuthMiddleware())
	router.GET("/", func(c *gin.Context) {
		value, _ := c.Get(api.SessionCookieValueKey)
		c.String(http.StatusOK, value.(string))
	})

	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: api.SessionCookieName, Value: "session-token"})
	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if response.Body.String() != "session-token" {
		t.Fatalf("expected session token to be stored, got %q", response.Body.String())
	}
}

func TestSessionAuthenticatorRequiresSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	input := &openapi3filter.AuthenticationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req},
		SecuritySchemeName:     "sessionCookie",
	}

	if err := SessionAuthenticator(context.Background(), input); !errors.Is(err, ErrSessionCookieRequired) {
		t.Fatalf("expected ErrSessionCookieRequired, got %v", err)
	}
}

func TestSessionAuthenticatorAcceptsSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: api.SessionCookieName, Value: "session-token"})
	input := &openapi3filter.AuthenticationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req},
		SecuritySchemeName:     "sessionCookie",
	}

	if err := SessionAuthenticator(context.Background(), input); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
