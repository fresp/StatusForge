package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	subcomponentservice "github.com/fresp/StatusForge/internal/services/subcomponent"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetSubComponents(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var componentID string
		if cid := c.Param("id"); cid != "" {
			if _, err := primitive.ObjectIDFromHex(cid); err == nil {
				componentID = cid
			}
		}
		if cid := c.Query("componentId"); cid != "" {
			if _, err := primitive.ObjectIDFromHex(cid); err == nil {
				componentID = cid
			}
		}

		service := subcomponentservice.NewService(repository.NewMongoSubComponentRepository(db))
		subs, total64, err := service.List(ctx, componentID, page, limit)
		if err != nil {
			writeDomainError(c, err)
			return
		}
		if subs == nil {
			subs = []models.SubComponent{}
		}
		writePaginatedResponse(c, subs, int(total64), page, limit)
	}
}

func CreateSubComponent(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ComponentID string                 `json:"componentId" binding:"required"`
			Name        string                 `json:"name" binding:"required"`
			Description string                 `json:"description"`
			Status      models.ComponentStatus `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		compID, err := primitive.ObjectIDFromHex(req.ComponentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid componentId"})
			return
		}

		if req.Status == "" {
			req.Status = models.StatusOperational
		}

		sub := models.SubComponent{
			ID:          primitive.NewObjectID(),
			ComponentID: compID,
			Name:        req.Name,
			Description: req.Description,
			Status:      req.Status,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := subcomponentservice.NewService(repository.NewMongoSubComponentRepository(db))
		sub, err = service.Create(ctx, subcomponentservice.CreateInput{
			ComponentID: req.ComponentID,
			Name:        req.Name,
			Description: req.Description,
			Status:      req.Status,
		})
		if err != nil {
			writeDomainError(c, err)
			return
		}
		c.JSON(http.StatusCreated, sub)
	}
}

func UpdateSubComponent(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Status      models.ComponentStatus `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := subcomponentservice.NewService(repository.NewMongoSubComponentRepository(db))
		sub, err := service.Update(ctx, id, subcomponentservice.UpdateInput{
			Name:        req.Name,
			Description: req.Description,
			Status:      req.Status,
		})
		if err != nil {
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusOK, sub)
	}
}
