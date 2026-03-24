// Package server provides the unified server functionality.
package server

import (
	"context"
	"errors"
	"log"
	"strings"
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

	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("[SERVER] Database initialization failed: %v", err)
	}

	setupPending := !cfg.SetupDone
	runtimeMongoReady := cfg.DBEngine == "mongodb" && database.GetDB() != nil

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
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	RegisterSetupRoutes(r)

	if runtimeMongoReady {
		RegisterAPIRoutes(r, hub, cfg, db)
	} else {
		r.NoRoute(setupFallbackHandler())
	}

	// Register health check endpoint
	r.GET("/health", HealthCheckHandler())

	// If Worker is enabled, start it
	if cfg.EnableWorker && runtimeMongoReady {
		log.Println("[SERVER] Starting monitoring worker...")
		go StartWorker(ctx, db, rdb)
	} else if cfg.EnableWorker && !runtimeMongoReady {
		if setupPending {
			log.Println("[SERVER] Worker deferred: setup is not complete")
		} else {
			log.Println("[SERVER] Worker disabled: selected database runtime is not yet available")
		}
	} else {
		log.Println("[SERVER] Worker disabled via ENABLE_WORKER=false")
	}

	if runtimeMongoReady {
		SeedAdmin(db, cfg)
	}

	if !runtimeMongoReady {
		if db != nil {
			log.Printf("[SERVER] Ignoring unexpected DB state while setup mode is active")
		}
	} else {
		// Serve React root
		r.NoRoute(StaticFileServer())
	}

	// Setup shutdown signal handler - this will call cancel() when SIGTERM/SIGINT is received
	SetupShutdownSignalHandler(cancel)

	log.Printf("[SERVER] Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("[SERVER] Failed to run server: %v", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Gracefully stop worker if it was started
	if cfg.EnableWorker && runtimeMongoReady {
		log.Println("[SERVER] Stopping worker...")
		if err := StopWorker(); err != nil {
			log.Printf("[SERVER] Error stopping worker: %v", err)
		}
		// Wait for all worker goroutines to finish
		time.Sleep(1 * time.Second)
	}

	log.Println("[SERVER] Server shutdown complete")
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	return nil
}

func setupFallbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/api/") {
			liveCfg := configs.Load()
			if !liveCfg.SetupDone {
				c.JSON(503, gin.H{"error": "setup required", "setupDone": false})
				return
			}
			status := database.BuildStatus(liveCfg)
			c.JSON(503, gin.H{
				"error":            "selected database runtime is not yet available",
				"setupDone":        liveCfg.SetupDone,
				"engine":           liveCfg.DBEngine,
				"runtimeSupported": status.RuntimeSupported,
			})
			return
		}

		StaticFileServer()(c)
	}
}
