package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	shared "github.com/fresp/StatusForge/internal/domain/shared"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubMonitorLogsService struct {
	result    models.PaginatedResult[models.MonitorLog]
	err       error
	called    bool
	page      int
	limit     int
	monitorID primitive.ObjectID
}

func (s *stubMonitorLogsService) GetMonitorLogsPaginated(_ context.Context, monitorID primitive.ObjectID, page, limit int) (models.PaginatedResult[models.MonitorLog], error) {
	s.called = true
	s.monitorID = monitorID
	s.page = page
	s.limit = limit
	if s.err != nil {
		return models.PaginatedResult[models.MonitorLog]{}, s.err
	}
	return s.result, nil
}

func TestGetMonitorLogsHandlerReturnsPaginatedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	monitorID := primitive.NewObjectID()
	checkedAt := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	stub := &stubMonitorLogsService{
		result: models.PaginatedResult[models.MonitorLog]{
			Items: []models.MonitorLog{{
				ID:        primitive.NewObjectID(),
				MonitorID: monitorID,
				Status:    models.MonitorUp,
				CheckedAt: checkedAt,
			}},
			Page:       2,
			Total:      35,
			TotalPages: 4,
		},
	}

	r := gin.New()
	r.GET("/api/monitors/:id/logs", getMonitorLogsWithService(stub))

	req := httptest.NewRequest(http.MethodGet, "/api/monitors/"+monitorID.Hex()+"/logs?page=2&limit=10", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.True(t, stub.called)
	assert.Equal(t, monitorID, stub.monitorID)
	assert.Equal(t, 2, stub.page)
	assert.Equal(t, 10, stub.limit)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	assert.Equal(t, float64(2), payload["page"])
	assert.Equal(t, float64(35), payload["total"])
	assert.Equal(t, float64(4), payload["total_pages"])

	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "up", item["status"])
}

func TestGetMonitorLogsHandlerRejectsInvalidPaginationParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	monitorID := primitive.NewObjectID()
	stub := &stubMonitorLogsService{}

	r := gin.New()
	r.GET("/api/monitors/:id/logs", getMonitorLogsWithService(stub))

	req := httptest.NewRequest(http.MethodGet, "/api/monitors/"+monitorID.Hex()+"/logs?page=0", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(t, `{"error":"invalid page query parameter"}`, resp.Body.String())
	assert.False(t, stub.called)
}

func TestGetMonitorLogsHandlerRejectsInvalidMonitorID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &stubMonitorLogsService{}

	r := gin.New()
	r.GET("/api/monitors/:id/logs", getMonitorLogsWithService(stub))

	req := httptest.NewRequest(http.MethodGet, "/api/monitors/not-a-valid-id/logs", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(t, `{"error":"invalid monitor id"}`, resp.Body.String())
	assert.False(t, stub.called)
}

func TestGetMonitorLogsHandlerMapsServiceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	monitorID := primitive.NewObjectID()

	t.Run("not found", func(t *testing.T) {
		stub := &stubMonitorLogsService{err: errors.Join(shared.ErrNotFound, errors.New("monitor logs not found"))}
		r := gin.New()
		r.GET("/api/monitors/:id/logs", getMonitorLogsWithService(stub))

		req := httptest.NewRequest(http.MethodGet, "/api/monitors/"+monitorID.Hex()+"/logs", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		require.Equal(t, http.StatusNotFound, resp.Code)
		assert.Contains(t, resp.Body.String(), "not found")
	})

	t.Run("unexpected", func(t *testing.T) {
		stub := &stubMonitorLogsService{err: errors.New("db down")}
		r := gin.New()
		r.GET("/api/monitors/:id/logs", getMonitorLogsWithService(stub))

		req := httptest.NewRequest(http.MethodGet, "/api/monitors/"+monitorID.Hex()+"/logs", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		require.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"error":"internal server error"}`, resp.Body.String())
	})
}
