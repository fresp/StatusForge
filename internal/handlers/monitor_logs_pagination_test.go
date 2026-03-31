package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestParseMonitorLogsPaginationParamsDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/monitors/abc/logs", nil)
	c.Request = req

	page, limit, err := parseMonitorLogsPaginationParams(c)

	assert.NoError(t, err)
	assert.Equal(t, 1, page)
	assert.Equal(t, 10, limit)
}

func TestParseMonitorLogsPaginationParamsAcceptsExplicitValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/monitors/abc/logs?page=2&limit=50", nil)
	c.Request = req

	page, limit, err := parseMonitorLogsPaginationParams(c)

	assert.NoError(t, err)
	assert.Equal(t, 2, page)
	assert.Equal(t, 50, limit)
}

func TestParseMonitorLogsPaginationParamsClampsLimitAt100(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/monitors/abc/logs?page=1&limit=500", nil)
	c.Request = req

	page, limit, err := parseMonitorLogsPaginationParams(c)

	assert.NoError(t, err)
	assert.Equal(t, 1, page)
	assert.Equal(t, 100, limit)
}

func TestParseMonitorLogsPaginationParamsRejectsInvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/monitors/abc/logs?page=zero", nil)
	c.Request = req

	_, _, err := parseMonitorLogsPaginationParams(c)

	assert.Error(t, err)
	assert.Equal(t, "invalid page query parameter", err.Error())
}

func TestParseMonitorLogsPaginationParamsRejectsInvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/monitors/abc/logs?limit=0", nil)
	c.Request = req

	_, _, err := parseMonitorLogsPaginationParams(c)

	assert.Error(t, err)
	assert.Equal(t, "invalid limit query parameter", err.Error())
}
