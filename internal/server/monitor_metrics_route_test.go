package server

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fresp/Statora/configs"
	"github.com/fresp/Statora/internal/handlers"
)

func TestMonitorMetricsRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	hub := handlers.NewHub()
	cfg := &configs.Config{JWTSecret: "test-secret"}

	RegisterAPIRoutes(router, hub, cfg)

	found := false
	for _, route := range router.Routes() {
		if route.Method == "GET" && route.Path == "/api/v1/monitors/:id/metrics" {
			found = true
			break
		}
	}

	assert.True(t, found, "v1 monitor metrics route should be registered")
}
