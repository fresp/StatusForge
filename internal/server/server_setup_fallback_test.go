package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetupFallbackHandler_RequiresSetupWhenUnset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DB_ENGINE", "")

	r := gin.New()
	r.NoRoute(setupFallbackHandler())

	req, _ := http.NewRequest(http.MethodGet, "/api/settings/status-page", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "setup required")
	assert.Contains(t, w.Body.String(), `"setupDone":false`)
}

func TestSetupFallbackHandler_ReflectsUpdatedSetupState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DB_ENGINE", "")

	r := gin.New()
	r.NoRoute(setupFallbackHandler())

	firstReq, _ := http.NewRequest(http.MethodGet, "/api/settings/status-page", nil)
	firstRes := httptest.NewRecorder()
	r.ServeHTTP(firstRes, firstReq)
	assert.Equal(t, http.StatusServiceUnavailable, firstRes.Code)
	assert.Contains(t, firstRes.Body.String(), "setup required")

	t.Setenv("DB_ENGINE", "sqlite")

	secondReq, _ := http.NewRequest(http.MethodGet, "/api/settings/status-page", nil)
	secondRes := httptest.NewRecorder()
	r.ServeHTTP(secondRes, secondReq)

	assert.Equal(t, http.StatusServiceUnavailable, secondRes.Code)
	assert.Contains(t, secondRes.Body.String(), "selected database runtime is not yet available")
	assert.Contains(t, secondRes.Body.String(), `"setupDone":true`)
	assert.Contains(t, secondRes.Body.String(), `"engine":"sqlite"`)
	assert.Contains(t, secondRes.Body.String(), `"runtimeSupported":false`)
	assert.NotContains(t, secondRes.Body.String(), "setup required")
}
