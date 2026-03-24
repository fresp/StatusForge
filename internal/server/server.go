// Package server provides the unified server functionality.
package server

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/fresp/StatusForge/configs"
	"github.com/fresp/StatusForge/internal/database"
	"github.com/fresp/StatusForge/internal/handlers"
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

	// Context setup (conditional graceful)
	var ctx context.Context
	var cancel context.CancelFunc

	if cfg.GracefulShutdown {
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()

		// Only setup signal handler if graceful enabled
		SetupShutdownSignalHandler(cancel, time.Duration(cfg.GracefulTimeout)*time.Second)
	} else {
		ctx = context.Background()
	}

	db := database.GetDB()
	rdb := database.GetRedis()

	// Create WebSocket hub
	hub := handlers.NewHub()
	go hub.Run()

	// Setup API routes with Gin
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

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
	r.NoRoute(StaticFileServer())

	log.Printf("[SERVER] Server starting on port %s", cfg.Port)

	// Run server (blocking)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("[SERVER] Failed to run server: %v", err)
	}

	// ONLY run graceful shutdown logic if enabled
	if cfg.GracefulShutdown {
		// Wait for shutdown signal
		<-ctx.Done()

		// Gracefully stop worker
		if cfg.EnableWorker {
			log.Println("[SERVER] Stopping worker...")
			if err := StopWorker(); err != nil {
				log.Printf("[SERVER] Error stopping worker: %v", err)
			}
			time.Sleep(1 * time.Second)
		}

		log.Println("[SERVER] Server shutdown complete")
	}

	return nil
}