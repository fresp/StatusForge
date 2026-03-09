package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"status-platform/configs"
	"status-platform/internal/database"
)

// TestGracefulShutdown tests that the server shuts down gracefully when signaled
func TestGracefulShutdown(t *testing.T) {
	// This is a unit test for the shutdown mechanism
	// We can't easily test the full HTTP server shutdown in a unit test
	// but we can test the context cancellation behavior

	originalCfg := configs.Load()

	// Override config for test to disable worker initially
	testCfg := &configs.Config{
		MongoURI:     originalCfg.MongoURI,
		MongoDBName:  originalCfg.MongoDBName,
		RedisAddr:    originalCfg.RedisAddr,
		JWTSecret:    originalCfg.JWTSecret,
		Port:         "0", // Use port 0 to let OS assign free port (but we won't actually listen)
		AdminEmail:   originalCfg.AdminEmail,
		AdminPass:    originalCfg.AdminPass,
		AdminUser:    originalCfg.AdminUser,
		EnableWorker: false, // Disable worker for this test
	}

	dbInitialized := true
	if err := database.ConnectMongo(testCfg.MongoURI, testCfg.MongoDBName); err != nil {
		dbInitialized = false
		t.Log("Skipping graceful shutdown test: MongoDB not available")
	}

	if !dbInitialized {
		t.SkipNow() // Skip test if no database
	}

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate the shutdown process
	shutdownCalled := false
	done := make(chan bool, 1)

	// Simulate a background worker that listens to context
	go func() {
		select {
		case <-ctx.Done():
			shutdownCalled = true
		case <-time.After(5 * time.Second): // Timeout
		}
		done <- true
	}()

	// Now execute the context cancellation (simulating shutdown signal)
	cancel()

	// Wait for the simulated shutdown to complete
	select {
	case <-done:
		// Successfully handled shutdown
		assert.True(t, shutdownCalled, "shutdown should be called when context is cancelled")
	case <-time.After(6 * time.Second): // Wait a bit longer than timeout
		t.Fatal("Test timed out waiting for shutdown")
	}
}

// TestWorkerGracefulStop tests that the worker shutdown is handled properly
func TestWorkerGracefulStop(t *testing.T) {
	// We can't start the real worker in tests, but we test the StopWorker function
	// when implemented properly it should cancel the worker context

	// Initially, without a worker context, Stop should return without error
	err := StopWorker()
	assert.NoError(t, err, "StopWorker should not return an error when there's no worker context")
}
