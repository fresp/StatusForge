package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fresp/Statora/configs"
	"github.com/fresp/Statora/internal/handlers"
	"github.com/fresp/Statora/internal/middleware"
)

func TestAdminOnlyRoutesForbidOperatorRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())
	hub := handlers.NewHub()

	cfg := &configs.Config{JWTSecret: "test-rbac-secret"}
	RegisterAPIRoutes(router, hub, cfg)

	token, err := middleware.GenerateTokenWithClaims(middleware.TokenClaimsInput{
		UserID:      "operator-id",
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
		{name: "GET /api/users", method: http.MethodGet, path: "/api/users"},
		{name: "PATCH /api/users/:id", method: http.MethodPatch, path: "/api/users/507f1f77bcf86cd799439011"},
		{name: "POST /api/users/invitations", method: http.MethodPost, path: "/api/users/invitations"},
		{name: "GET /api/users/invitations", method: http.MethodGet, path: "/api/users/invitations"},
		{name: "POST /api/users/invitations/:id/refresh", method: http.MethodPost, path: "/api/users/invitations/507f1f77bcf86cd799439011/refresh"},
		{name: "DELETE /api/users/invitations/:id", method: http.MethodDelete, path: "/api/users/invitations/507f1f77bcf86cd799439011"},
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

func TestPartialTokenCanOnlyAccessMeAndMFAEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())
	hub := handlers.NewHub()

	cfg := &configs.Config{JWTSecret: "test-rbac-secret"}
	RegisterAPIRoutes(router, hub, cfg)

	token, err := middleware.GenerateTokenWithClaims(middleware.TokenClaimsInput{
		UserID:      "admin-id",
		Username:    "admin-user",
		Role:        "admin",
		MFAVerified: false,
		Secret:      cfg.JWTSecret,
	})
	assert.NoError(t, err)

	allowedPartialRoutes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "GET /api/auth/me", method: http.MethodGet, path: "/api/auth/me"},
		{name: "POST /api/auth/mfa/setup", method: http.MethodPost, path: "/api/auth/mfa/setup", body: `{}`},
		{name: "POST /api/auth/mfa/verify", method: http.MethodPost, path: "/api/auth/mfa/verify", body: `{"code":"123456"}`},
		{name: "POST /api/auth/mfa/recovery/verify", method: http.MethodPost, path: "/api/auth/mfa/recovery/verify", body: `{"code":"AAAAA-BBBBB"}`},
		{name: "POST /api/auth/mfa/disable", method: http.MethodPost, path: "/api/auth/mfa/disable", body: `{"password":"secret123","code":"123456"}`},
	}

	for _, route := range allowedPartialRoutes {
		t.Run(route.name, func(t *testing.T) {
			req, reqErr := http.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
			assert.NoError(t, reqErr)
			req.Header.Set("Authorization", "Bearer "+token)
			if route.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.NotEqual(t, http.StatusForbidden, resp.Code)
		})
	}

	restrictedRoutes := []struct {
		name   string
		method string
		path   string
	}{
		{name: "GET /api/incidents", method: http.MethodGet, path: "/api/incidents"},
		{name: "GET /api/users", method: http.MethodGet, path: "/api/users"},
	}

	for _, route := range restrictedRoutes {
		t.Run(route.name, func(t *testing.T) {
			req, reqErr := http.NewRequest(route.method, route.path, nil)
			assert.NoError(t, reqErr)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusForbidden, resp.Code)
			assert.Contains(t, resp.Body.String(), "mfa verification required")
		})
	}
}

func TestVerifiedTokenCanAccessRoleProtectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())
	hub := handlers.NewHub()

	cfg := &configs.Config{JWTSecret: "test-rbac-secret"}
	RegisterAPIRoutes(router, hub, cfg)

	verifiedAdminToken, err := middleware.GenerateTokenWithClaims(middleware.TokenClaimsInput{
		UserID:      "admin-id",
		Username:    "admin-user",
		Role:        "admin",
		MFAVerified: true,
		Secret:      cfg.JWTSecret,
	})
	assert.NoError(t, err)

	verifiedOperatorToken, err := middleware.GenerateTokenWithClaims(middleware.TokenClaimsInput{
		UserID:      "operator-id",
		Username:    "operator-user",
		Role:        "operator",
		MFAVerified: true,
		Secret:      cfg.JWTSecret,
	})
	assert.NoError(t, err)

	adminReq, adminReqErr := http.NewRequest(http.MethodGet, "/api/users", nil)
	assert.NoError(t, adminReqErr)
	adminReq.Header.Set("Authorization", "Bearer "+verifiedAdminToken)

	adminResp := httptest.NewRecorder()
	router.ServeHTTP(adminResp, adminReq)
	assert.NotEqual(t, http.StatusForbidden, adminResp.Code)

	operatorReq, operatorReqErr := http.NewRequest(http.MethodGet, "/api/incidents", nil)
	assert.NoError(t, operatorReqErr)
	operatorReq.Header.Set("Authorization", "Bearer "+verifiedOperatorToken)

	operatorResp := httptest.NewRecorder()
	router.ServeHTTP(operatorResp, operatorReq)
	assert.NotEqual(t, http.StatusForbidden, operatorResp.Code)
}
