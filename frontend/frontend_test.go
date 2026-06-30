package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesRendersIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/game/chug.value.funds", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `data-game-id="chug.value.funds"`) {
		t.Fatalf("expected rendered game id, got %q", response.Body.String())
	}
}

func TestRegisterRoutesServesAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), "Nobody is Perfect") {
		t.Fatalf("expected frontend script, got %q", response.Body.String())
	}
}
