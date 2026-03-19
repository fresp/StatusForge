package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"status-platform/configs"
	"status-platform/internal/handlers"
	"status-platform/internal/middleware"
)

func TestAdminOnlyRoutesForbidOperatorRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	hub := handlers.NewHub()

	cfg := &configs.Config{JWTSecret: "test-rbac-secret"}
	RegisterAPIRoutes(router, hub, cfg)

	token, err := middleware.GenerateTokenWithClaims(middleware.TokenClaimsInput{
		AdminID:     "operator-id",
		Username:    "operator-user",
		Role:        "operator",
		MFAVerified: true,
		Secret:      cfg.JWTSecret,
	})
	assert.NoError(t, err)

	adminOnlyRoutes := []struct {
		name   string
		method string
		path   string
	}{
		{name: "POST /api/components", method: http.MethodPost, path: "/api/components"},
		{name: "PATCH /api/components/:id", method: http.MethodPatch, path: "/api/components/507f1f77bcf86cd799439011"},
		{name: "DELETE /api/components/:id", method: http.MethodDelete, path: "/api/components/507f1f77bcf86cd799439011"},
		{name: "POST /api/subcomponents", method: http.MethodPost, path: "/api/subcomponents"},
		{name: "PATCH /api/subcomponents/:id", method: http.MethodPatch, path: "/api/subcomponents/507f1f77bcf86cd799439011"},
		{name: "GET /api/monitors", method: http.MethodGet, path: "/api/monitors"},
		{name: "POST /api/monitors", method: http.MethodPost, path: "/api/monitors"},
		{name: "POST /api/monitors/test", method: http.MethodPost, path: "/api/monitors/test"},
		{name: "PUT /api/monitors/:id", method: http.MethodPut, path: "/api/monitors/507f1f77bcf86cd799439011"},
		{name: "DELETE /api/monitors/:id", method: http.MethodDelete, path: "/api/monitors/507f1f77bcf86cd799439011"},
		{name: "GET /api/monitors/:id/logs", method: http.MethodGet, path: "/api/monitors/507f1f77bcf86cd799439011/logs"},
		{name: "GET /api/monitors/:id/uptime", method: http.MethodGet, path: "/api/monitors/507f1f77bcf86cd799439011/uptime"},
		{name: "GET /api/monitors/:id/history", method: http.MethodGet, path: "/api/monitors/507f1f77bcf86cd799439011/history"},
		{name: "GET /api/monitors/outages", method: http.MethodGet, path: "/api/monitors/outages"},
		{name: "GET /api/subscribers", method: http.MethodGet, path: "/api/subscribers"},
		{name: "DELETE /api/subscribers/:id", method: http.MethodDelete, path: "/api/subscribers/507f1f77bcf86cd799439011"},
		{name: "GET /api/admins", method: http.MethodGet, path: "/api/admins"},
		{name: "PATCH /api/admins/:id", method: http.MethodPatch, path: "/api/admins/507f1f77bcf86cd799439011"},
		{name: "POST /api/admins/invitations", method: http.MethodPost, path: "/api/admins/invitations"},
		{name: "GET /api/admins/invitations", method: http.MethodGet, path: "/api/admins/invitations"},
		{name: "POST /api/admins/invitations/:id/refresh", method: http.MethodPost, path: "/api/admins/invitations/507f1f77bcf86cd799439011/refresh"},
		{name: "DELETE /api/admins/invitations/:id", method: http.MethodDelete, path: "/api/admins/invitations/507f1f77bcf86cd799439011"},
	}

	for _, route := range adminOnlyRoutes {
		t.Run(route.name, func(t *testing.T) {
			req, reqErr := http.NewRequest(route.method, route.path, nil)
			assert.NoError(t, reqErr)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusForbidden, resp.Code)
		})
	}
}
