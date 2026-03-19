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
	api.POST("/admins/invitations/activate", handlers.ActivateAdminInvitation(database.GetDB()))

	auth := api.Group("")
	auth.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	auth.GET("/auth/me", handlers.GetMe(database.GetDB()))

	adminOnly := auth.Group("")
	adminOnly.Use(middleware.RequireRoles("admin"))

	incidentAndMaintenance := auth.Group("")
	incidentAndMaintenance.Use(middleware.RequireRoles("admin", "operator"))

	incidentAndMaintenance.GET("/incidents", handlers.GetIncidents(database.GetDB()))
	incidentAndMaintenance.POST("/incidents", handlers.CreateIncident(database.GetDB(), hub))
	incidentAndMaintenance.PATCH("/incidents/:id", handlers.UpdateIncident(database.GetDB(), hub))
	incidentAndMaintenance.POST("/incidents/:id/update", handlers.AddIncidentUpdate(database.GetDB(), hub))
	incidentAndMaintenance.GET("/incidents/:id/updates", handlers.GetIncidentUpdates(database.GetDB()))

	incidentAndMaintenance.GET("/maintenance", handlers.GetMaintenance(database.GetDB()))
	incidentAndMaintenance.POST("/maintenance", handlers.CreateMaintenance(database.GetDB()))
	incidentAndMaintenance.PATCH("/maintenance/:id", handlers.UpdateMaintenance(database.GetDB()))

	incidentAndMaintenance.GET("/components", handlers.GetComponents(database.GetDB()))
	incidentAndMaintenance.GET("/components/:id/subcomponents", handlers.GetSubComponents(database.GetDB()))
	incidentAndMaintenance.GET("/subcomponents", handlers.GetSubComponents(database.GetDB()))

	adminOnly.POST("/components", handlers.CreateComponent(database.GetDB(), hub))
	adminOnly.PATCH("/components/:id", handlers.UpdateComponent(database.GetDB(), hub))
	adminOnly.DELETE("/components/:id", handlers.DeleteComponent(database.GetDB()))

	adminOnly.POST("/subcomponents", handlers.CreateSubComponent(database.GetDB()))
	adminOnly.PATCH("/subcomponents/:id", handlers.UpdateSubComponent(database.GetDB()))

	adminOnly.GET("/monitors", handlers.GetMonitors(database.GetDB()))
	adminOnly.POST("/monitors", handlers.CreateMonitor(database.GetDB()))
	adminOnly.POST("/monitors/test", handlers.TestMonitor())
	adminOnly.PUT("/monitors/:id", handlers.UpdateMonitor(database.GetDB()))
	adminOnly.DELETE("/monitors/:id", handlers.DeleteMonitor(database.GetDB()))
	adminOnly.GET("/monitors/:id/logs", handlers.GetMonitorLogs(database.GetDB()))
	adminOnly.GET("/monitors/:id/uptime", handlers.GetMonitorUptime(database.GetDB()))
	adminOnly.GET("/monitors/:id/history", handlers.GetMonitorHistory(database.GetDB()))
	adminOnly.GET("/monitors/outages", handlers.GetMonitorOutages(database.GetDB()))

	adminOnly.GET("/subscribers", handlers.GetSubscribers(database.GetDB()))
	adminOnly.DELETE("/subscribers/:id", handlers.DeleteSubscriber(database.GetDB()))

	adminOnly.GET("/admins", handlers.GetAdmins(database.GetDB()))
	adminOnly.PATCH("/admins/:id", handlers.PatchAdmin(database.GetDB()))
	adminOnly.POST("/admins/invitations", handlers.CreateAdminInvitation(database.GetDB()))
	adminOnly.GET("/admins/invitations", handlers.GetAdminInvitations(database.GetDB()))
	adminOnly.POST("/admins/invitations/:id/refresh", handlers.RefreshAdminInvitation(database.GetDB()))
	adminOnly.DELETE("/admins/invitations/:id", handlers.RevokeAdminInvitation(database.GetDB()))
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
		Role:         "admin",
		Status:       "active",
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	if _, err := db.Collection("admins").InsertOne(ctx, admin); err != nil {
		log.Printf("[HTTP] Failed to seed admin: %v", err)
		return
	}

	log.Printf("[HTTP] Admin seeded: %s / %s", cfg.AdminEmail, cfg.AdminUser)
}
