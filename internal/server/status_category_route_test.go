package server

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fresp/Statora/configs"
	"github.com/fresp/Statora/internal/handlers"
)

func TestStatusCategoryRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	hub := handlers.NewHub()
	cfg := &configs.Config{JWTSecret: "test-secret"}

	RegisterAPIRoutes(router, hub, cfg)

	routes := router.Routes()
	foundStatusCategory := false
	foundV1StatusCategory := false

	for _, route := range routes {
		if route.Method == "GET" && route.Path == "/api/status/category/:prefix" {
			foundStatusCategory = true
		}
		if route.Method == "GET" && route.Path == "/api/v1/status/category/:prefix" {
			foundV1StatusCategory = true
		}
	}

	assert.True(t, foundStatusCategory, "public status category route should be registered")
	assert.True(t, foundV1StatusCategory, "v1 status category route should be registered")
}
