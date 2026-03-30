package handlers

import (
	"context"
	"errors"
	"time"

	shared "github.com/fresp/StatusForge/internal/domain/shared"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	subscriberservice "github.com/fresp/StatusForge/internal/services/subscriber"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
)

func Subscribe(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := subscriberservice.NewService(repository.NewMongoSubscriberRepository(db))
		sub, err := service.Create(ctx, req.Email)
		if err != nil {
			if isConflictError(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "email already subscribed"})
				return
			}
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "subscribed successfully", "id": sub.ID})
	}
}

func GetSubscribers(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := subscriberservice.NewService(repository.NewMongoSubscriberRepository(db))
		subs, total64, err := service.List(ctx, page, limit)
		if err != nil {
			writeDomainError(c, err)
			return
		}
		if subs == nil {
			subs = []models.Subscriber{}
		}
		writePaginatedResponse(c, subs, int(total64), page, limit)
	}
}

func DeleteSubscriber(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := subscriberservice.NewService(repository.NewMongoSubscriberRepository(db))
		err = service.DeleteByID(ctx, id)
		if err != nil {
			if isNotFoundError(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "subscriber not found"})
				return
			}
			writeDomainError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "unsubscribed"})
	}
}

func isConflictError(err error) bool {
	return errors.Is(err, shared.ErrConflict)
}

func isNotFoundError(err error) bool {
	return errors.Is(err, shared.ErrNotFound)
}
