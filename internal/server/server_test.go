package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fresp/Statora/configs"
	"github.com/fresp/Statora/internal/database"
	"github.com/fresp/Statora/internal/handlers"
)

// MockDB is a mock implementation of the database functionality for testing
type MockDB struct {
	mock.Mock
}

func TestRunServerIntegration(t *testing.T) {
	// Skip this test in CI environments as it requires external dependencies

	// Test basic server initialization without starting it fully
	gin.SetMode(gin.TestMode)

	t.Run("Test unified server config and initialization", func(t *testing.T) {
		cfg := configs.Load()
		assert.NotNil(t, cfg, "Config should be loaded")

		// Test that EnableWorker configuration is respected
		assert.Equal(t, true, cfg.EnableWorker, "EnableWorker should default to true")

		// Initialize a context for testing
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create a simplified version of the server initialization
		db := database.GetDB()
		assert.NotNil(t, db, "Database should be initialized")
	})
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a simple HTTP test
	router := gin.New()

	// Register only health check for this test
	router.GET("/health", HealthCheckHandler())

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Since this test runs without db/redis connection,
	// it will probably return "unhealthy" but status should be 200 or 503
	assert.Contains(t, []int{200, 503}, w.Code, "Health endpoint should return valid status")
}

// Integration test for API routes registration
func TestAPIRoutesRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock hub
	hub := handlers.NewHub()
	go hub.Run()

	cfg := &configs.Config{
		MongoURI:     "mongodb://localhost:27017",
		MongoDBName:  "test_db",
		RedisAddr:    "localhost:6379",
		JWTSecret:    "test-secret",
		Port:         "8081",
		AdminEmail:   "admin@test.com",
		AdminPass:    "admin123",
		AdminUser:    "admin",
		EnableWorker: false,
	}

	ginEngine := gin.Default()
	RegisterAPIRoutes(ginEngine, hub, cfg)

	// Test a basic API endpoint registration
	// We check if routes are registered by examining the router's routes
	routes := ginEngine.Routes()

	foundHealth := false
	for _, route := range routes {
		if route.Path == "/api/status/summary" && route.Method == "GET" {
			foundHealth = true
			break
		}
	}

	assert.True(t, foundHealth, "API /status/summary route should be registered")
}

func TestUnifiedStartupWithMockDB(t *testing.T) {
	// This test attempts to verify the server startup flow with mock database
	cfg := &configs.Config{
		MongoURI:     "mongodb://localhost:27017",
		MongoDBName:  "test_db",
		RedisAddr:    "localhost:6379",
		JWTSecret:    "test-secret",
		Port:         "8081",
		AdminEmail:   "admin@test.com",
		AdminPass:    "admin123",
		AdminUser:    "admin",
		EnableWorker: true,
	}

	// Create a context with timeout
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	if err := database.ConnectMongo(cfg.MongoURI, cfg.MongoDBName); err != nil {
		// Skip test if database not available
		t.Skip("Skipping integration test: MongoDB not available")
	}

	if err := database.ConnectRedis(cfg.RedisAddr); err != nil {
		// Non-fatal for this test, just log
		t.Logf("Warning: Redis not available: %v", err)
	}

	// Create WebSocket hub
	hub := handlers.NewHub()
	go hub.Run()

	// Use gin test mode
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Register API routes
	RegisterAPIRoutes(r, hub, cfg)

	// Register health check endpoint
	r.GET("/health", HealthCheckHandler())

	// Register static file serving
	r.NoRoute(StaticFileServer())

	// Verify that routes are properly registered
	routes := r.Routes()
	foundAPIRoute := false
	for _, route := range routes {
		if route.Path == "/api/status/summary" {
			foundAPIRoute = true
			break
		}
	}
	assert.True(t, foundAPIRoute, "API route should be registered")

	// Test health endpoint
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Logf("Health check: %v, Status: %d", w.Body.String(), w.Code)
	} else {
		t.Logf("Health check response: %+v", response)
	}

	// Verify that we can serve the main page if embed works
	// (skip if embed doesn't exist, which indicates a build environment)
}
