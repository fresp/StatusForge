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

func GetIncidents(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		filter := bson.M{}
		if status := c.Query("status"); status == "active" {
			filter["status"] = bson.M{"$ne": models.IncidentResolved}
		} else if status != "" {
			filter["status"] = status
		}

		cursor, err := db.Collection("incidents").Find(ctx, filter,
			options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var incidents []models.Incident
		if err := cursor.All(ctx, &incidents); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if incidents == nil {
			incidents = []models.Incident{}
		}
		c.JSON(http.StatusOK, incidents)
	}
}

func CreateIncident(db *mongo.Database, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Title              string                `json:"title" binding:"required"`
			Description        string                `json:"description"`
			Status             models.IncidentStatus `json:"status"`
			Impact             models.IncidentImpact `json:"impact"`
			AffectedComponents []string              `json:"affectedComponents"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Status == "" {
			req.Status = models.IncidentInvestigating
		}
		if req.Impact == "" {
			req.Impact = models.ImpactMinor
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
		for _, s := range req.AffectedComponents {
			oid, err := primitive.ObjectIDFromHex(s)
			if err == nil {
				compIDs = append(compIDs, oid)
			}
		}
		if compIDs == nil {
			compIDs = []primitive.ObjectID{}
		}

		incident := models.Incident{
			ID:                 primitive.NewObjectID(),
			Title:              req.Title,
			Description:        req.Description,
			Status:             req.Status,
			Impact:             req.Impact,
			CreatorID:          &userID,
			CreatorUsername:    creatorName,
			AffectedComponents: compIDs,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := db.Collection("incidents").InsertOne(ctx, incident); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		DispatchWebhookEvent(db, "incident_created", incident)
		BroadcastEvent(hub, "incident_created", incident)
		c.JSON(http.StatusCreated, incident)
	}
}

func UpdateIncident(db *mongo.Database, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Title              string                `json:"title"`
			Description        string                `json:"description"`
			Status             models.IncidentStatus `json:"status"`
			Impact             models.IncidentImpact `json:"impact"`
			AffectedComponents []string              `json:"affectedComponents"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		setFields := bson.M{"updatedAt": time.Now()}
		if req.Title != "" {
			setFields["title"] = req.Title
		}
		if req.Description != "" {
			setFields["description"] = req.Description
		}
		if req.Status != "" {
			setFields["status"] = req.Status
			if req.Status == models.IncidentResolved {
				now := time.Now()
				setFields["resolvedAt"] = now
			}
		}
		if req.Impact != "" {
			setFields["impact"] = req.Impact
		}
		if len(req.AffectedComponents) > 0 {
			var compIDs []primitive.ObjectID
			for _, s := range req.AffectedComponents {
				oid, err := primitive.ObjectIDFromHex(s)
				if err == nil {
					compIDs = append(compIDs, oid)
				}
			}
			setFields["affectedComponents"] = compIDs
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var incident models.Incident
		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		err = db.Collection("incidents").FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": setFields}, opts).Decode(&incident)
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		eventType := "incident_updated"
		if incident.Status == models.IncidentResolved {
			eventType = "incident_resolved"
		}
		DispatchWebhookEvent(db, eventType, incident)
		BroadcastEvent(hub, eventType, incident)
		c.JSON(http.StatusOK, incident)
	}
}

func AddIncidentUpdate(db *mongo.Database, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		incidentID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident id"})
			return
		}

		var req struct {
			Message string                `json:"message" binding:"required"`
			Status  models.IncidentStatus `json:"status" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		update := models.IncidentUpdate{
			ID:         primitive.NewObjectID(),
			IncidentID: incidentID,
			Message:    req.Message,
			Status:     req.Status,
			CreatedAt:  time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// insert update log
		if _, err := db.Collection("incident_updates").InsertOne(ctx, update); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// build update fields
		updateFields := bson.M{
			"status":    req.Status,
			"updatedAt": time.Now(),
		}

		// if resolved, set resolvedAt
		if req.Status == models.IncidentResolved {
			updateFields["resolvedAt"] = time.Now()
		}

		// update incident document
		_, err = db.Collection("incidents").UpdateOne(
			ctx,
			bson.M{"_id": incidentID},
			bson.M{"$set": updateFields},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		DispatchWebhookEvent(db, "incident_update_added", update)
		BroadcastEvent(hub, "incident_update_added", update)

		c.JSON(http.StatusCreated, update)
	}
}

func GetIncidentUpdates(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		incidentID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := db.Collection("incident_updates").Find(ctx,
			bson.M{"incidentId": incidentID},
			options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var updates []models.IncidentUpdate
		if err := cursor.All(ctx, &updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if updates == nil {
			updates = []models.IncidentUpdate{}
		}
		c.JSON(http.StatusOK, updates)
	}
}
