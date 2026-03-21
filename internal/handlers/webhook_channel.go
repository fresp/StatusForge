package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetWebhookChannels retrieves all webhook channels from the database.
func GetWebhookChannels(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := db.Collection("webhook_channels").Find(ctx, bson.M{},
			options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var channels []models.WebhookChannel
		if err := cursor.All(ctx, &channels); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if channels == nil {
			channels = []models.WebhookChannel{}
		}

		c.JSON(http.StatusOK, channels)
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

		channel := models.WebhookChannel{
			ID:        primitive.NewObjectID(),
			Name:      req.Name,
			URL:       req.URL,
			Enabled:   true,
			CreatedAt: time.Now(),
		}

		if _, err := db.Collection("webhook_channels").InsertOne(ctx, channel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

		res, err := db.Collection("webhook_channels").DeleteOne(ctx, bson.M{"_id": id})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if res.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "webhook channel not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "webhook channel deleted"})
	}
}
