package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"status-platform/internal/models"
	"status-platform/internal/utils"
)

func GetMonitors(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := db.Collection("monitors").Find(ctx, bson.M{},
			options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var monitors []models.Monitor
		if err := cursor.All(ctx, &monitors); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if monitors == nil {
			monitors = []models.Monitor{}
		}
		c.JSON(http.StatusOK, monitors)
	}
}

func CreateMonitor(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name            string             `json:"name" binding:"required"`
			Type            models.MonitorType `json:"type" binding:"required"`
			Target          string             `json:"target" binding:"required"`
			IntervalSeconds int                `json:"intervalSeconds"`
			TimeoutSeconds  int                `json:"timeoutSeconds"`
			ComponentID     string             `json:"componentId"`
			SubComponentID  string             `json:"subComponentId"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Must have at least one target
		if req.ComponentID == "" && req.SubComponentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "must specify componentId or subComponentId"})
			return
		}

		if req.IntervalSeconds == 0 {
			req.IntervalSeconds = 60
		}

		if req.TimeoutSeconds == 0 {
			req.TimeoutSeconds = 30
		}

		var compID primitive.ObjectID
		var subCompID primitive.ObjectID

		// If subcomponent is provided, prioritize it
		if req.SubComponentID != "" && req.SubComponentID != "000000000000000000000000" {
			oid, err := primitive.ObjectIDFromHex(req.SubComponentID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subComponentId"})
				return
			}
			subCompID = oid
		} else if req.ComponentID != "" && req.ComponentID != "000000000000000000000000" {
			oid, err := primitive.ObjectIDFromHex(req.ComponentID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid componentId"})
				return
			}
			compID = oid
		}

		monitor := models.Monitor{
			ID:              primitive.NewObjectID(),
			Name:            req.Name,
			Type:            req.Type,
			Target:          req.Target,
			IntervalSeconds: req.IntervalSeconds,
			TimeoutSeconds:  req.TimeoutSeconds,
			ComponentID:     compID,
			SubComponentID:  subCompID,
			CreatedAt:       time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := db.Collection("monitors").InsertOne(ctx, monitor); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, monitor)
	}
}

func UpdateMonitor(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Name            string             `json:"name" binding:"required"`
			Type            models.MonitorType `json:"type" binding:"required"`
			Target          string             `json:"target" binding:"required"`
			IntervalSeconds int                `json:"intervalSeconds"`
			TimeoutSeconds  int                `json:"timeoutSeconds"`
			ComponentID     string             `json:"componentId"`
			SubComponentID  string             `json:"subComponentId"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.ComponentID == "" && req.SubComponentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "must specify componentId or subComponentId"})
			return
		}

		if req.IntervalSeconds == 0 {
			req.IntervalSeconds = 60
		}

		if req.TimeoutSeconds == 0 {
			req.TimeoutSeconds = 30
		}

		var compID primitive.ObjectID
		var subCompID primitive.ObjectID

		if req.SubComponentID != "" && req.SubComponentID != "000000000000000000000000" {
			oid, err := primitive.ObjectIDFromHex(req.SubComponentID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subComponentId"})
				return
			}
			subCompID = oid
		} else if req.ComponentID != "" && req.ComponentID != "000000000000000000000000" {
			oid, err := primitive.ObjectIDFromHex(req.ComponentID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid componentId"})
				return
			}
			compID = oid
		}

		update := bson.M{
			"$set": bson.M{
				"name":            req.Name,
				"type":            req.Type,
				"target":          req.Target,
				"intervalSeconds": req.IntervalSeconds,
				"timeoutSeconds":  req.TimeoutSeconds,
				"componentId":     compID,
				"subComponentId":  subCompID,
				"updatedAt":       time.Now(),
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := db.Collection("monitors").UpdateOne(ctx, bson.M{"_id": id}, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if res.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "updated"})
	}
}

func TestMonitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Type           models.MonitorType `json:"type" binding:"required"`
			Target         string             `json:"target" binding:"required"`
			TimeoutSeconds int                `json:"timeoutSeconds"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		timeout := time.Duration(req.TimeoutSeconds) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		start := time.Now()
		status := models.MonitorUp
		statusCode := 0

		switch req.Type {
		case models.MonitorHTTP:
			code, err := utils.CheckHTTP(req.Target, timeout)
			statusCode = code
			if err != nil || code >= 500 || code == 0 {
				status = models.MonitorDown
			}
		case models.MonitorTCP:
			if err := utils.CheckTCP(req.Target, timeout); err != nil {
				status = models.MonitorDown
			}
		case models.MonitorDNS:
			if err := utils.CheckDNS(req.Target, timeout); err != nil {
				status = models.MonitorDown
			}
		case models.MonitorPing:
			if err := utils.CheckPing(req.Target, timeout); err != nil {
				status = models.MonitorDown
			}
		}

		responseTime := time.Since(start).Milliseconds()

		c.JSON(http.StatusOK, gin.H{
			"status":       status,
			"statusCode":   statusCode,
			"responseTime": responseTime,
		})
	}
}

func GetMonitorLogs(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		monitorID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		limit := int64(100)
		cursor, err := db.Collection("monitor_logs").Find(ctx,
			bson.M{"monitorId": monitorID},
			options.Find().SetSort(bson.D{{Key: "checkedAt", Value: -1}}).SetLimit(limit))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var logs []models.MonitorLog
		if err := cursor.All(ctx, &logs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if logs == nil {
			logs = []models.MonitorLog{}
		}
		c.JSON(http.StatusOK, logs)
	}
}

func GetMonitorUptime(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		monitorID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		since := time.Now().AddDate(0, 0, -90)
		cursor, err := db.Collection("daily_uptime").Find(ctx,
			bson.M{"monitorId": monitorID, "date": bson.M{"$gte": since}},
			options.Find().SetSort(bson.D{{Key: "date", Value: 1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var uptime []models.DailyUptime
		if err := cursor.All(ctx, &uptime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if uptime == nil {
			uptime = []models.DailyUptime{}
		}
		c.JSON(http.StatusOK, uptime)
	}
}

func DeleteMonitor(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := db.Collection("monitors").DeleteOne(ctx, bson.M{"_id": id})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if res.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

func GetMonitorOutages(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sortField := bson.D{{Key: "startedAt", Value: -1}}
		cursor, err := db.Collection("outages").Find(ctx, bson.M{},
			options.Find().SetSort(sortField))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var outages []models.Outage
		if err := cursor.All(ctx, &outages); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if outages == nil {
			outages = []models.Outage{}
		}
		c.JSON(http.StatusOK, outages)
	}
}

func GetMonitorHistory(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		monitorID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		limit := int64(100)
		cursor, err := db.Collection("enhanced_monitor_logs").Find(ctx,
			bson.M{"monitorId": monitorID},
			options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var logs []models.EnhancedMonitorLog
		if err := cursor.All(ctx, &logs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if logs == nil {
			logs = []models.EnhancedMonitorLog{}
		}
		c.JSON(http.StatusOK, logs)
	}
}
