package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fresp/Statora/internal/models"
	statusservice "github.com/fresp/Statora/internal/services/status"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubMonitorMetricsBuilder struct {
	metrics *statusservice.ServiceMetrics
	err     error
	called  bool
	id      primitive.ObjectID
	now     time.Time
}

type stubMonitorLookup struct {
	monitor *statusserviceMonitor
	err     error
	called  bool
	id      primitive.ObjectID
}

type statusserviceMonitor struct {
	ComponentID    primitive.ObjectID
	SubComponentID primitive.ObjectID
}

func (s *stubMonitorLookup) FindMonitorByID(_ context.Context, id primitive.ObjectID) (*models.Monitor, error) {
	s.called = true
	s.id = id
	if s.err != nil {
		return nil, s.err
	}
	if s.monitor == nil {
		return nil, nil
	}
	return &models.Monitor{ComponentID: s.monitor.ComponentID, SubComponentID: s.monitor.SubComponentID}, nil
}

func (s *stubMonitorLookup) FindMonitorBySubComponentID(ctx context.Context, id primitive.ObjectID) (*models.Monitor, error) {
	return s.FindMonitorByID(ctx, id)
}

func (s *stubMonitorMetricsBuilder) BuildServiceMetrics(_ context.Context, serviceID primitive.ObjectID, now time.Time) (*statusservice.ServiceMetrics, error) {
	s.called = true
	s.id = serviceID
	s.now = now
	if s.err != nil {
		return nil, s.err
	}
	return s.metrics, nil
}

func TestGetMonitorMetricsReturnsBadRequestForInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	builder := &stubMonitorMetricsBuilder{}
	r := gin.New()
	r.GET("/api/v1/monitors/:id/metrics", getMonitorMetricsWithBuilder(builder, func() time.Time {
		return time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/not-an-object-id/metrics", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.False(t, builder.called)
}

func TestGetMonitorMetricsReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	builder := &stubMonitorMetricsBuilder{err: errors.New("boom")}
	r := gin.New()
	r.GET("/api/v1/monitors/:id/metrics", getMonitorMetricsWithBuilder(builder, func() time.Time {
		return time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	}))

	id := primitive.NewObjectID()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/"+id.Hex()+"/metrics", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.True(t, builder.called)
	assert.Equal(t, id, builder.id)
	assert.JSONEq(t, `{"error":"failed to build monitor metrics"}`, resp.Body.String())
}

func TestGetMonitorMetricsReturnsExpectedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fixedNow := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	builder := &stubMonitorMetricsBuilder{
		metrics: &statusservice.ServiceMetrics{
			Latency: &statusservice.ServiceLatencyMetrics{P90: 900, P99: 1000},
			Availability: &statusservice.ServiceAvailabilityMetrics{
				Last30Days: 80,
			},
			History: []statusservice.ServiceMetricsHistoryEntry{{
				Month:        "March 2026",
				Latency:      statusservice.ServiceLatencyMetrics{P90: 900, P99: 1000},
				Availability: 80,
			}},
		},
	}

	r := gin.New()
	r.GET("/api/v1/monitors/:id/metrics", getMonitorMetricsWithBuilder(builder, func() time.Time {
		return fixedNow
	}))

	id := primitive.NewObjectID()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/"+id.Hex()+"/metrics", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, builder.called)
	assert.Equal(t, id, builder.id)
	assert.Equal(t, fixedNow, builder.now)

	var decoded statusservice.ServiceMetrics
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &decoded))
	require.NotNil(t, decoded.Latency)
	require.NotNil(t, decoded.Availability)
	assert.Equal(t, float64(900), decoded.Latency.P90)
	assert.Equal(t, float64(1000), decoded.Latency.P99)
	assert.Equal(t, float64(80), decoded.Availability.Last30Days)
	require.Len(t, decoded.History, 1)
	assert.Equal(t, "March 2026", decoded.History[0].Month)
}

func TestGetMonitorMetricsResolvesServiceIDFromMonitorLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fixedNow := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	componentID := primitive.NewObjectID()
	builder := &stubMonitorMetricsBuilder{metrics: &statusservice.ServiceMetrics{History: []statusservice.ServiceMetricsHistoryEntry{}}}
	lookup := &stubMonitorLookup{monitor: &statusserviceMonitor{ComponentID: componentID}}

	r := gin.New()
	r.GET("/api/v1/monitors/:id/metrics", getMonitorMetricsWithBuilderAndLookup(builder, lookup, func() time.Time { return fixedNow }))

	monitorID := primitive.NewObjectID()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/"+monitorID.Hex()+"/metrics", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.True(t, lookup.called)
	require.Equal(t, monitorID, lookup.id)
	require.True(t, builder.called)
	assert.Equal(t, componentID, builder.id)
	assert.Equal(t, fixedNow, builder.now)
}

func TestGetMonitorMetricsReturnsNotFoundWhenMonitorMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	builder := &stubMonitorMetricsBuilder{metrics: &statusservice.ServiceMetrics{History: []statusservice.ServiceMetricsHistoryEntry{}}}
	lookup := &stubMonitorLookup{}

	r := gin.New()
	r.GET("/api/v1/monitors/:id/metrics", getMonitorMetricsWithBuilderAndLookup(builder, lookup, time.Now))

	monitorID := primitive.NewObjectID()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/"+monitorID.Hex()+"/metrics", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
	assert.JSONEq(t, `{"error":"monitor not found"}`, resp.Body.String())
	assert.False(t, builder.called)
}

func TestGetMonitorMetricsReturnsInternalServerErrorWhenLookupFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	builder := &stubMonitorMetricsBuilder{metrics: &statusservice.ServiceMetrics{History: []statusservice.ServiceMetricsHistoryEntry{}}}
	lookup := &stubMonitorLookup{err: errors.New("lookup failed")}

	r := gin.New()
	r.GET("/api/v1/monitors/:id/metrics", getMonitorMetricsWithBuilderAndLookup(builder, lookup, time.Now))

	monitorID := primitive.NewObjectID()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/"+monitorID.Hex()+"/metrics", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(t, `{"error":"failed to resolve monitor metrics"}`, resp.Body.String())
	assert.False(t, builder.called)
}
