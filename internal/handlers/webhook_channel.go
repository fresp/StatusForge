package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	webhookservice "github.com/fresp/StatusForge/internal/services/webhook"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetWebhookChannels retrieves all webhook channels from the database.
func GetWebhookChannels(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := webhookservice.NewService(repository.NewMongoWebhookChannelRepository(db))
		channels, total64, err := service.List(ctx, page, limit)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		if channels == nil {
			channels = []models.WebhookChannel{}
		}

		writePaginatedResponse(c, channels, int(total64), page, limit)
	}
}

// CreateWebhookChannel creates a new webhook channel.
func CreateWebhookChannel(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
			URL  string `json:"url" binding:"required,url"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := webhookservice.NewService(repository.NewMongoWebhookChannelRepository(db))
		channel, err := service.Create(ctx, req.Name, req.URL)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusCreated, channel)
	}
}

// DeleteWebhookChannel deletes a webhook channel by ID.
func DeleteWebhookChannel(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := webhookservice.NewService(repository.NewMongoWebhookChannelRepository(db))
		err = service.DeleteByID(ctx, id)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "webhook channel deleted"})
	}
}
