package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/max0l/nobody-is-perfect-go/api"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
)

const SessionCookieName = "session"

var ErrSessionCookieRequired = errors.New("session cookie required")

func SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, protected := c.Get(api.SessionCookieScopes); !protected {
			c.Next()
			return
		}

		token, ok := sessionCookieValue(c.Request)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": api.UnauthorizedError})
			return
		}

		c.Set(api.SessionCookieValueKey, token)
		c.Next()
	}
}

func SessionAuthenticator(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	if input.SecuritySchemeName != "sessionCookie" {
		return fmt.Errorf("unsupported security scheme %q", input.SecuritySchemeName)
	}

	token, ok := sessionCookieValue(input.RequestValidationInput.Request)
	if !ok {
		return ErrSessionCookieRequired
	}

	if ginContext := ginmiddleware.GetGinContext(ctx); ginContext != nil {
		ginContext.Set(api.SessionCookieValueKey, token)
	}

	return nil
}

func sessionCookieValue(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}

	return cookie.Value, true
}
