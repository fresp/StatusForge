package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	incidentservice "github.com/fresp/StatusForge/internal/services/incident"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type incidentAffectedComponentInput struct {
	ComponentID     string   `json:"componentId"`
	SubComponentIDs []string `json:"subComponentIds"`
}

type incidentRequestBody struct {
	Title                    string                           `json:"title" binding:"required"`
	Description              string                           `json:"description"`
	Status                   models.IncidentStatus            `json:"status"`
	Impact                   models.IncidentImpact            `json:"impact"`
	AffectedComponents       []string                         `json:"affectedComponents"`
	Components               []string                         `json:"components"`
	AffectedComponentTargets []incidentAffectedComponentInput `json:"affectedComponentTargets"`
	AffectedComponentsNew    []incidentAffectedComponentInput `json:"affected_components"`
}

func invalidAffectedComponentsError() gin.H {
	return gin.H{"error": "invalid affected components payload"}
}

func invalidAffectedComponentsReferenceError() gin.H {
	return gin.H{"error": "one or more affected components or subcomponents are invalid"}
}

func mapIncidentTargets(in []incidentAffectedComponentInput) []incidentservice.AffectedComponentInput {
	out := make([]incidentservice.AffectedComponentInput, 0, len(in))
	for _, t := range in {
		out = append(out, incidentservice.AffectedComponentInput{
			ComponentID:     t.ComponentID,
			SubComponentIDs: t.SubComponentIDs,
		})
	}
	return out
}

func GetIncidents(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := incidentservice.NewService(repository.NewMongoIncidentRepository(db))
		incidents, total, err := service.List(ctx, c.Query("status"), c.Query("start_date"), c.Query("end_date"), page, limit)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		if incidents == nil {
			incidents = []models.Incident{}
		}
		writePaginatedResponse(c, incidents, int(total), page, limit)
	}
}

func CreateIncident(db *mongo.Database, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req incidentRequestBody
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

		creatorUsername, _ := c.Get("username")
		creatorName, _ := creatorUsername.(string)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := incidentservice.NewService(repository.NewMongoIncidentRepository(db))
		incident, err := service.Create(ctx, incidentservice.CreateInput{
			RequestBody: incidentservice.RequestBody{
				Title:                    req.Title,
				Description:              req.Description,
				Status:                   req.Status,
				Impact:                   req.Impact,
				AffectedComponents:       req.AffectedComponents,
				Components:               req.Components,
				AffectedComponentTargets: mapIncidentTargets(req.AffectedComponentTargets),
				AffectedComponentsNew:    mapIncidentTargets(req.AffectedComponentsNew),
			},
			CreatorIDHex:    userIDHex,
			CreatorUsername: creatorName,
		})
		if err != nil {
			if errors.Is(err, incidentservice.ErrInvalidAffectedComponentsPayload) {
				c.JSON(http.StatusBadRequest, invalidAffectedComponentsError())
				return
			}
			if errors.Is(err, incidentservice.ErrInvalidAffectedComponentsReference) {
				c.JSON(http.StatusBadRequest, invalidAffectedComponentsReferenceError())
				return
			}
			writeDomainError(c, err)
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

		var req incidentRequestBody
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := incidentservice.NewService(repository.NewMongoIncidentRepository(db))
		incident, err := service.Update(ctx, id, incidentservice.RequestBody{
			Title:                    req.Title,
			Description:              req.Description,
			Status:                   req.Status,
			Impact:                   req.Impact,
			AffectedComponents:       req.AffectedComponents,
			Components:               req.Components,
			AffectedComponentTargets: mapIncidentTargets(req.AffectedComponentTargets),
			AffectedComponentsNew:    mapIncidentTargets(req.AffectedComponentsNew),
		})
		if err != nil {
			if errors.Is(err, incidentservice.ErrInvalidAffectedComponentsPayload) {
				c.JSON(http.StatusBadRequest, invalidAffectedComponentsError())
				return
			}
			if errors.Is(err, incidentservice.ErrInvalidAffectedComponentsReference) {
				c.JSON(http.StatusBadRequest, invalidAffectedComponentsReferenceError())
				return
			}
			writeDomainError(c, err)
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

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := incidentservice.NewService(repository.NewMongoIncidentRepository(db))
		update, err := service.AddUpdate(ctx, incidentID, req.Message, req.Status)
		if err != nil {
			writeDomainError(c, err)
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

		service := incidentservice.NewService(repository.NewMongoIncidentRepository(db))
		updates, err := service.ListUpdates(ctx, incidentID)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		if updates == nil {
			updates = []models.IncidentUpdate{}
		}
		c.JSON(http.StatusOK, updates)
	}
}
