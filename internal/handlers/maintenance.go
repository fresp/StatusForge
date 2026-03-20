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

func GetMaintenance(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := db.Collection("maintenance").Find(ctx, bson.M{},
			options.Find().SetSort(bson.D{{Key: "startTime", Value: -1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var items []models.Maintenance
		if err := cursor.All(ctx, &items); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if items == nil {
			items = []models.Maintenance{}
		}
		c.JSON(http.StatusOK, items)
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

		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid startTime format, use RFC3339"})
			return
		}
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endTime format, use RFC3339"})
			return
		}

		rawUserID, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user context"})
			return
		}

		userIDHex, ok := rawUserID.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user context"})
			return
		}

		userID, err := primitive.ObjectIDFromHex(userIDHex)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user id"})
			return
		}

		creatorUsername, _ := c.Get("username")
		creatorName, _ := creatorUsername.(string)

		var compIDs []primitive.ObjectID
		for _, s := range req.Components {
			oid, err := primitive.ObjectIDFromHex(s)
			if err == nil {
				compIDs = append(compIDs, oid)
			}
		}
		if compIDs == nil {
			compIDs = []primitive.ObjectID{}
		}

		status := models.MaintenanceScheduled
		if time.Now().After(startTime) {
			status = models.MaintenanceInProgress
		}

		m := models.Maintenance{
			ID:              primitive.NewObjectID(),
			Title:           req.Title,
			Description:     req.Description,
			CreatorID:       &userID,
			CreatorUsername: creatorName,
			Components:      compIDs,
			StartTime:       startTime,
			EndTime:         endTime,
			Status:          status,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := db.Collection("maintenance").InsertOne(ctx, m); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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

		setFields := bson.M{}
		if req.Title != "" {
			setFields["title"] = req.Title
		}
		if req.Description != "" {
			setFields["description"] = req.Description
		}
		if req.Status != "" {
			setFields["status"] = req.Status
		}
		if req.StartTime != "" {
			t, err := time.Parse(time.RFC3339, req.StartTime)
			if err == nil {
				setFields["startTime"] = t
			}
		}
		if req.EndTime != "" {
			t, err := time.Parse(time.RFC3339, req.EndTime)
			if err == nil {
				setFields["endTime"] = t
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var m models.Maintenance
		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		err = db.Collection("maintenance").FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": setFields}, opts).Decode(&m)
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "maintenance not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, m)
	}
}
