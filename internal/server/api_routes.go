// Package server provides the unified server functionality.
package server

import (
	"context"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"status-platform/configs"
	"status-platform/internal/database"
	"status-platform/internal/handlers"
	"status-platform/internal/middleware"
	"status-platform/internal/models"
)

// RegisterAPIRoutes registers all API routes on the given Gin engine
func RegisterAPIRoutes(r *gin.Engine, hub *handlers.Hub, cfg *configs.Config) {
	// Apply CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")

	api.GET("/status/summary", handlers.GetStatusSummary(database.GetDB()))
	api.GET("/status/components", handlers.GetStatusComponents(database.GetDB()))
	api.GET("/status/incidents", handlers.GetStatusIncidents(database.GetDB()))
	api.POST("/subscribe", handlers.Subscribe(database.GetDB()))

	api.POST("/auth/login", handlers.Login(database.GetDB(), cfg.JWTSecret))

	auth := api.Group("")
	auth.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	auth.GET("/auth/me", handlers.GetMe(database.GetDB()))

	auth.GET("/components", handlers.GetComponents(database.GetDB()))
	auth.POST("/components", handlers.CreateComponent(database.GetDB(), hub))
	auth.PATCH("/components/:id", handlers.UpdateComponent(database.GetDB(), hub))
	auth.DELETE("/components/:id", handlers.DeleteComponent(database.GetDB()))

	auth.GET("/components/:id/subcomponents", handlers.GetSubComponents(database.GetDB()))
	auth.GET("/subcomponents", handlers.GetSubComponents(database.GetDB()))
	auth.POST("/subcomponents", handlers.CreateSubComponent(database.GetDB()))
	auth.PATCH("/subcomponents/:id", handlers.UpdateSubComponent(database.GetDB()))

	auth.GET("/monitors", handlers.GetMonitors(database.GetDB()))
	auth.POST("/monitors", handlers.CreateMonitor(database.GetDB()))
	auth.POST("/monitors/test", handlers.TestMonitor())
	auth.PUT("/monitors/:id", handlers.UpdateMonitor(database.GetDB()))
	auth.DELETE("/monitors/:id", handlers.DeleteMonitor(database.GetDB()))
	auth.GET("/monitors/:id/logs", handlers.GetMonitorLogs(database.GetDB()))
	auth.GET("/monitors/:id/uptime", handlers.GetMonitorUptime(database.GetDB()))
	auth.GET("/monitors/:id/history", handlers.GetMonitorHistory(database.GetDB()))
	auth.GET("/monitors/outages", handlers.GetMonitorOutages(database.GetDB()))

	auth.GET("/incidents", handlers.GetIncidents(database.GetDB()))
	auth.POST("/incidents", handlers.CreateIncident(database.GetDB(), hub))
	auth.PATCH("/incidents/:id", handlers.UpdateIncident(database.GetDB(), hub))
	auth.POST("/incidents/:id/update", handlers.AddIncidentUpdate(database.GetDB(), hub))
	auth.GET("/incidents/:id/updates", handlers.GetIncidentUpdates(database.GetDB()))

	auth.GET("/maintenance", handlers.GetMaintenance(database.GetDB()))
	auth.POST("/maintenance", handlers.CreateMaintenance(database.GetDB()))
	auth.PATCH("/maintenance/:id", handlers.UpdateMaintenance(database.GetDB()))

	auth.GET("/subscribers", handlers.GetSubscribers(database.GetDB()))
	auth.DELETE("/subscribers/:id", handlers.DeleteSubscriber(database.GetDB()))
}

func SeedAdmin(db *mongo.Database, cfg *configs.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var existing models.Admin
	if err := db.Collection("admins").FindOne(ctx, bson.M{"email": cfg.AdminEmail}).Decode(&existing); err == nil {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPass), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[HTTP] Failed to hash admin password: %v", err)
		return
	}

	admin := models.Admin{
		ID:           primitive.NewObjectID(),
		Username:     cfg.AdminUser,
		Email:        cfg.AdminEmail,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	if _, err := db.Collection("admins").InsertOne(ctx, admin); err != nil {
		log.Printf("[HTTP] Failed to seed admin: %v", err)
		return
	}

	log.Printf("[HTTP] Admin seeded: %s / %s", cfg.AdminEmail, cfg.AdminUser)
}
