package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/fresp/Statora/internal/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetComponents(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		coll := db.Collection("components")
		total64, err := coll.CountDocuments(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		skip := int64((page - 1) * limit)

		cursor, err := coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}).SetSkip(skip).SetLimit(int64(limit)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var components []models.Component
		if err := cursor.All(ctx, &components); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if components == nil {
			components = []models.Component{}
		}
		writePaginatedResponse(c, components, int(total64), page, limit)
	}
}

func CreateComponent(db *mongo.Database, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string                 `json:"name" binding:"required"`
			Description string                 `json:"description"`
			Status      models.ComponentStatus `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Status == "" {
			req.Status = models.StatusOperational
		}

		comp := models.Component{
			ID:          primitive.NewObjectID(),
			Name:        req.Name,
			Description: req.Description,
			Status:      req.Status,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := db.Collection("components").InsertOne(ctx, comp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		BroadcastEvent(hub, "component_created", comp)
		c.JSON(http.StatusCreated, comp)
	}
}

func UpdateComponent(db *mongo.Database, hub *Hub) gin.HandlerFunc {
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

		update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}
		if req.Name != "" {
			update["$set"].(bson.M)["name"] = req.Name
		}
		if req.Description != "" {
			update["$set"].(bson.M)["description"] = req.Description
		}
		if req.Status != "" {
			update["$set"].(bson.M)["status"] = req.Status
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var comp models.Component
		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		err = db.Collection("components").FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&comp)
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "component not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		BroadcastEvent(hub, "component_updated", comp)
		c.JSON(http.StatusOK, comp)
	}
}

func DeleteComponent(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := db.Collection("components").DeleteOne(ctx, bson.M{"_id": id})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if res.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "component not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}
