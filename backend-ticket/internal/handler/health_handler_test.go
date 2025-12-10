package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthHandler_Health(t *testing.T) {
	// Create handler with nil db (Health endpoint doesn't use db)
	handler := &HealthHandler{db: nil}

	router := gin.New()
	router.GET("/health", handler.Health)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !contains(body, "ticket-service") {
		t.Errorf("expected 'ticket-service' in response, got %s", body)
	}
	if !contains(body, "ok") {
		t.Errorf("expected 'ok' in response, got %s", body)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
