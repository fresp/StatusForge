package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestParsePaginationParamsDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	c.Request = req

	page, limit, err := parsePaginationParams(c)

	assert.NoError(t, err)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, limit)
}

func TestParsePaginationParamsRejectsInvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/test?page=abc", nil)
	c.Request = req

	_, _, err := parsePaginationParams(c)

	assert.Error(t, err)
	assert.Equal(t, "invalid page query parameter", err.Error())
}

func TestParsePaginationParamsRejectsInvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/test?limit=0", nil)
	c.Request = req

	_, _, err := parsePaginationParams(c)

	assert.Error(t, err)
	assert.Equal(t, "invalid limit query parameter", err.Error())
}

func TestWritePaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/test", func(c *gin.Context) {
		writePaginatedResponse(c, []string{"a", "b"}, 12, 2, 5)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	assert.NoError(t, err)

	assert.Equal(t, float64(12), payload["total"])
	assert.Equal(t, float64(2), payload["page"])
	assert.Equal(t, float64(3), payload["total_pages"])

	items, ok := payload["items"].([]any)
	assert.True(t, ok)
	assert.Len(t, items, 2)
}

func TestClampPageToTotalPagesKeepsPageWhenInRange(t *testing.T) {
	clamped := clampPageToTotalPages(2, 10, 35)

	assert.Equal(t, 2, clamped)
}

func TestClampPageToTotalPagesClampsToLastPageWhenOutOfRange(t *testing.T) {
	clamped := clampPageToTotalPages(9, 10, 35)

	assert.Equal(t, 4, clamped)
}

func TestClampPageToTotalPagesKeepsRequestedPageWhenTotalIsEmpty(t *testing.T) {
	clamped := clampPageToTotalPages(3, 10, 0)

	assert.Equal(t, 3, clamped)
}

func TestClampPageToTotalPagesNormalizesPageBelowOne(t *testing.T) {
	clamped := clampPageToTotalPages(0, 10, 35)

	assert.Equal(t, 1, clamped)
}

func TestParsePaginationParamsWithDefaultConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	c.Request = req

	page, limit, err := ParsePaginationParams(c, DefaultPaginationConfig())

	assert.NoError(t, err)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, limit)
}

func TestParsePaginationParamsWithMonitorLogsConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	c.Request = req

	page, limit, err := ParsePaginationParams(c, MonitorLogsPaginationConfig())

	assert.NoError(t, err)
	assert.Equal(t, 1, page)
	assert.Equal(t, 10, limit)
}

func TestParsePaginationParamsWithCustomMaxLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/test?limit=500", nil)
	c.Request = req

	config := PaginationConfig{
		DefaultLimit: 20,
		MaxLimit:     100,
	}
	page, limit, err := ParsePaginationParams(c, config)

	assert.NoError(t, err)
	assert.Equal(t, 1, page)
	assert.Equal(t, 100, limit)
}

func TestParsePaginationParamsWithNoMaxLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/test?limit=500", nil)
	c.Request = req

	config := PaginationConfig{
		DefaultLimit: 20,
		MaxLimit:     0,
	}
	page, limit, err := ParsePaginationParams(c, config)

	assert.NoError(t, err)
	assert.Equal(t, 1, page)
	assert.Equal(t, 500, limit)
}
