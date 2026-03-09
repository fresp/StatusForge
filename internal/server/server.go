// Package server provides the unified server functionality.
package server

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"status-platform/configs"
	"status-platform/internal/database"
	"status-platform/internal/handlers"
)

// RunServer starts the unified server with API, Worker, and Web functionality
func RunServer() error {
	godotenv.Load()

	cfg := configs.Load()

	// Connect to databases
	if err := database.ConnectMongo(cfg.MongoURI, cfg.MongoDBName); err != nil {
		log.Fatalf("[SERVER] MongoDB connection failed: %v", err)
	}
	if err := database.ConnectRedis(cfg.RedisAddr); err != nil {
		log.Printf("[SERVER] Redis connection warning: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := database.GetDB()
	rdb := database.GetRedis()

	// Create WebSocket hub
	hub := handlers.NewHub()
	go hub.Run()

	// Setup API routes with Gin
	r := gin.Default()

	// Register API routes
	RegisterAPIRoutes(r, hub, cfg)

	// Register health check endpoint
	r.GET("/health", HealthCheckHandler())

	// If Worker is enabled, start it
	if cfg.EnableWorker {
		log.Println("[SERVER] Starting monitoring worker...")
		go StartWorker(ctx, db, rdb)
	} else {
		log.Println("[SERVER] Worker disabled via ENABLE_WORKER=false")
	}

	// Seed admin user
	SeedAdmin(db, cfg)


	// Serve React root
	r.GET("/", StaticFileServer())


	// Register static file serving (embedded React frontend) on all routes that don't match API
	r.NoRoute(StaticFileServer())

	// Setup shutdown signal handler - this will call cancel() when SIGTERM/SIGINT is received
	SetupShutdownSignalHandler(cancel)

	log.Printf("[SERVER] Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("[SERVER] Failed to run server: %v", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Gracefully stop worker if it was started
	if cfg.EnableWorker {
		log.Println("[SERVER] Stopping worker...")
		if err := StopWorker(); err != nil {
			log.Printf("[SERVER] Error stopping worker: %v", err)
		}
		// Wait for all worker goroutines to finish
		time.Sleep(1 * time.Second)
	}

	log.Println("[SERVER] Server shutdown complete")
	return nil
}
