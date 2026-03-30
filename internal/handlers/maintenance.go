package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	shared "github.com/fresp/StatusForge/internal/domain/shared"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	maintenanceservice "github.com/fresp/StatusForge/internal/services/maintenance"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetMaintenance(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := maintenanceservice.NewService(repository.NewMongoMaintenanceRepository(db))
		items, total, err := service.List(ctx, page, limit)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		if items == nil {
			items = []models.Maintenance{}
		}

		writePaginatedResponse(c, items, int(total), page, limit)
	}
}

func GetPublicMaintenance(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := maintenanceservice.NewService(repository.NewMongoMaintenanceRepository(db))
		items, total, err := service.ListPublic(ctx, page, limit)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		if items == nil {
			items = []models.Maintenance{}
		}

		writePaginatedResponse(c, items, int(total), page, limit)
	}
}

func CreateMaintenance(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Title       string   `json:"title" binding:"required"`
			Description string   `json:"description"`
			Components  []string `json:"components"`
			StartTime   string   `json:"startTime" binding:"required"`
			EndTime     string   `json:"endTime" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rawUserID, exists := c.Get("userId")
		if !exists {
			writeDomainError(c, fmt.Errorf("%w: missing authenticated user context", shared.ErrUnauthorized))
			return
		}

		userIDHex, ok := rawUserID.(string)
		if !ok {
			writeDomainError(c, fmt.Errorf("%w: invalid authenticated user context", shared.ErrUnauthorized))
			return
		}

		creatorUsername, _ := c.Get("username")
		creatorName, _ := creatorUsername.(string)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := maintenanceservice.NewService(repository.NewMongoMaintenanceRepository(db))
		m, err := service.Create(ctx, maintenanceservice.CreateInput{
			Title:           req.Title,
			Description:     req.Description,
			Components:      req.Components,
			StartTime:       req.StartTime,
			EndTime:         req.EndTime,
			CreatorIDHex:    userIDHex,
			CreatorUsername: creatorName,
		})
		if err != nil {
			writeDomainError(c, err)
			return
		}

		DispatchWebhookEvent(db, "maintenance_created", m)
		c.JSON(http.StatusCreated, m)
	}
}

func UpdateMaintenance(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Title       string                   `json:"title"`
			Description string                   `json:"description"`
			Status      models.MaintenanceStatus `json:"status"`
			StartTime   string                   `json:"startTime"`
			EndTime     string                   `json:"endTime"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := maintenanceservice.NewService(repository.NewMongoMaintenanceRepository(db))
		m, err := service.Update(ctx, id, maintenanceservice.UpdateInput{
			Title:       req.Title,
			Description: req.Description,
			Status:      req.Status,
			StartTime:   req.StartTime,
			EndTime:     req.EndTime,
		})
		if err != nil {
			writeDomainError(c, err)
			return
		}

		DispatchWebhookEvent(db, "maintenance_updated", m)
		c.JSON(http.StatusOK, m)
	}
}
