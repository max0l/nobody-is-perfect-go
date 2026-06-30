package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersMiddlewareSetsDefaultHeaders(t *testing.T) {
	response := securityHeadersResponse(t, "/", false, "")
	headers := response.Header()

	if got := headers.Get("Content-Security-Policy"); got != appContentSecurityPolicy {
		t.Fatalf("expected app CSP %q, got %q", appContentSecurityPolicy, got)
	}
	assertHeader(t, headers, "X-Content-Type-Options", "nosniff")
	assertHeader(t, headers, "X-Frame-Options", "DENY")
	assertHeader(t, headers, "Referrer-Policy", "no-referrer")
	assertHeader(t, headers, "X-XSS-Protection", "0")
	assertHeader(t, headers, "Cross-Origin-Opener-Policy", "same-origin")
	assertHeader(t, headers, "Cross-Origin-Resource-Policy", "same-origin")
	if got := headers.Get("Permissions-Policy"); !strings.Contains(got, "camera=()") || !strings.Contains(got, "microphone=()") {
		t.Fatalf("expected restrictive Permissions-Policy, got %q", got)
	}
	if got := headers.Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("expected no HSTS on HTTP request, got %q", got)
	}
}

func TestSecurityHeadersMiddlewareUsesSwaggerCSP(t *testing.T) {
	response := securityHeadersResponse(t, "/swagger/index.html", false, "")

	if got := response.Header().Get("Content-Security-Policy"); got != swaggerContentSecurityPolicy {
		t.Fatalf("expected swagger CSP %q, got %q", swaggerContentSecurityPolicy, got)
	}
}

func TestSecurityHeadersMiddlewareSetsHSTSForHTTPS(t *testing.T) {
	response := securityHeadersResponse(t, "/", true, "")
	assertHeader(t, response.Header(), "Strict-Transport-Security", "max-age=31536000; includeSubDomains")
}

func TestSecurityHeadersMiddlewareSetsHSTSForForwardedHTTPS(t *testing.T) {
	response := securityHeadersResponse(t, "/", false, "https")
	assertHeader(t, response.Header(), "Strict-Transport-Security", "max-age=31536000; includeSubDomains")
}

func securityHeadersResponse(t *testing.T, path string, tls bool, forwardedProto string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if tls {
		request = httptest.NewRequest(http.MethodGet, "https://example.test"+path, nil)
	}
	if forwardedProto != "" {
		request.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	router.ServeHTTP(response, request)

	return response
}

func assertHeader(t *testing.T, headers http.Header, name string, want string) {
	t.Helper()
	if got := headers.Get(name); got != want {
		t.Fatalf("expected %s %q, got %q", name, want, got)
	}
}
