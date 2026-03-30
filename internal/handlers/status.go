package handlers

import (
	"context"
	"net/http"
	"time"

	statusdomain "github.com/fresp/StatusForge/internal/domain/status"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	statusservice "github.com/fresp/StatusForge/internal/services/status"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type derivedStatusResult struct {
	ComponentStatus map[primitive.ObjectID]models.ComponentStatus
	SubStatus       map[primitive.ObjectID]models.ComponentStatus
}

type componentAggregationState struct {
	HasDirectImpact bool
	DirectStatus    models.ComponentStatus
	ImpactedSubIDs  map[primitive.ObjectID]struct{}
}

func maxStatus(a, b models.ComponentStatus) models.ComponentStatus {
	return statusdomain.MaxComponentStatus(a, b)
}

func mapIncidentImpactToStatus(impact models.IncidentImpact) models.ComponentStatus {
	switch impact {
	case models.ImpactCritical:
		return models.StatusMajorOutage
	case models.ImpactMajor:
		return models.StatusPartialOutage
	case models.ImpactMinor:
		return models.StatusDegradedPerf
	default:
		return models.StatusOperational
	}
}

func partialAggregateStatusFromSubStatus(status models.ComponentStatus) models.ComponentStatus {
	switch status {
	case models.StatusMajorOutage:
		return models.StatusPartialOutage
	case models.StatusPartialOutage:
		return models.StatusPartialOutage
	default:
		return status
	}
}

func deriveStatuses(
	components []models.Component,
	subs []models.SubComponent,
	activeIncidents []models.Incident,
	activeMaintenance []models.Maintenance,
) (derivedStatusResult, error) {
	componentStatus := make(map[primitive.ObjectID]models.ComponentStatus, len(components))
	subStatus := make(map[primitive.ObjectID]models.ComponentStatus, len(subs))

	for _, comp := range components {
		componentStatus[comp.ID] = comp.Status
	}
	for _, sub := range subs {
		subStatus[sub.ID] = sub.Status
	}

	subsByComp := make(map[primitive.ObjectID][]primitive.ObjectID)
	for _, sub := range subs {
		subsByComp[sub.ComponentID] = append(subsByComp[sub.ComponentID], sub.ID)
	}

	aggByComponent := make(map[primitive.ObjectID]*componentAggregationState, len(components))
	for _, comp := range components {
		aggByComponent[comp.ID] = &componentAggregationState{
			DirectStatus:   models.StatusOperational,
			ImpactedSubIDs: map[primitive.ObjectID]struct{}{},
		}
	}

	for _, inc := range activeIncidents {
		incidentStatus := mapIncidentImpactToStatus(inc.Impact)
		if incidentStatus == models.StatusOperational {
			continue
		}

		targets := normalizeIncidentTargetsForExpansion(inc)
		for _, target := range targets {
			state, exists := aggByComponent[target.ComponentID]
			if !exists {
				continue
			}

			if len(target.SubComponentIDs) == 0 {
				state.HasDirectImpact = true
				state.DirectStatus = maxStatus(state.DirectStatus, incidentStatus)
				for _, subID := range subsByComp[target.ComponentID] {
					state.ImpactedSubIDs[subID] = struct{}{}
					subStatus[subID] = maxStatus(subStatus[subID], incidentStatus)
				}
				continue
			}

			for _, subID := range target.SubComponentIDs {
				state.ImpactedSubIDs[subID] = struct{}{}
				subStatus[subID] = maxStatus(subStatus[subID], incidentStatus)
			}
		}
	}

	for _, maintenance := range activeMaintenance {
		for _, componentID := range maintenance.Components {
			state, exists := aggByComponent[componentID]
			if !exists {
				continue
			}

			state.HasDirectImpact = true
			state.DirectStatus = maxStatus(state.DirectStatus, models.StatusMaintenance)

			for _, subID := range subsByComp[componentID] {
				state.ImpactedSubIDs[subID] = struct{}{}
				subStatus[subID] = maxStatus(subStatus[subID], models.StatusMaintenance)
			}
		}
	}

	for _, comp := range components {
		state := aggByComponent[comp.ID]
		if state == nil {
			continue
		}

		totalSubs := len(subsByComp[comp.ID])
		impactedSubCount := len(state.ImpactedSubIDs)

		if !state.HasDirectImpact && impactedSubCount == 0 {
			continue
		}

		derived := state.DirectStatus
		if !state.HasDirectImpact {
			derived = models.StatusOperational
		}

		worstSubStatus := models.StatusOperational
		for subID := range state.ImpactedSubIDs {
			worstSubStatus = maxStatus(worstSubStatus, subStatus[subID])
		}

		if impactedSubCount > 0 && impactedSubCount < totalSubs {
			derived = maxStatus(derived, partialAggregateStatusFromSubStatus(worstSubStatus))
		} else {
			derived = maxStatus(derived, worstSubStatus)
		}

		componentStatus[comp.ID] = derived
	}

	return derivedStatusResult{
		ComponentStatus: componentStatus,
		SubStatus:       subStatus,
	}, nil
}

func normalizeIncidentTargetsForExpansion(incident models.Incident) []models.IncidentAffectedComponent {
	if len(incident.AffectedComponentTargets) > 0 {
		return incident.AffectedComponentTargets
	}

	targets := make([]models.IncidentAffectedComponent, 0, len(incident.AffectedComponents))
	for _, componentID := range incident.AffectedComponents {
		targets = append(targets, models.IncidentAffectedComponent{ComponentID: componentID})
	}

	return targets
}

type ComponentWithSubs struct {
	models.Component
	SubComponents []models.SubComponent `json:"subComponents"`
	UptimeHistory []UptimeBar           `json:"uptimeHistory"`
}

type UptimeBar struct {
	Date          string  `json:"date"`
	UptimePercent float64 `json:"uptimePercent"`
	Status        string  `json:"status"`
}

type StatusSummary struct {
	OverallStatus   string         `json:"overallStatus"`
	ComponentCounts map[string]int `json:"componentCounts"`
	ActiveIncidents int            `json:"activeIncidents"`
	ScheduledMaint  int            `json:"scheduledMaintenance"`
}

type ComponentWithSubStatusInfo struct {
	models.Component
	SubComponents []models.SubComponent `json:"subComponents"`
	UptimeHistory []UptimeBar           `json:"uptimeHistory"`
	LastIncident  *IncidentStatusInfo   `json:"lastIncident,omitempty"`
}

type IncidentStatusInfo struct {
	Date     time.Time `json:"date"`
	Duration string    `json:"duration"`
	Title    string    `json:"title"`
}

func GetStatusSummary(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		now := time.Now()
		service := statusservice.NewService(repository.NewMongoStatusRepository(db))
		summary, err := service.BuildSummary(ctx, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, StatusSummary{
			OverallStatus:   summary.OverallStatus,
			ComponentCounts: summary.ComponentCounts,
			ActiveIncidents: summary.ActiveIncidents,
			ScheduledMaint:  summary.ScheduledMaintenance,
		})
	}
}

func GetStatusComponents(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		now := time.Now()
		service := statusservice.NewService(repository.NewMongoStatusRepository(db))
		components, err := service.BuildComponents(ctx, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result := make([]ComponentWithSubStatusInfo, 0, len(components))
		for _, component := range components {
			uptimeHistory := make([]UptimeBar, 0, len(component.UptimeHistory))
			for _, bar := range component.UptimeHistory {
				uptimeHistory = append(uptimeHistory, UptimeBar{
					Date:          bar.Date,
					UptimePercent: bar.UptimePercent,
					Status:        bar.Status,
				})
			}

			var lastIncident *IncidentStatusInfo
			if component.LastIncident != nil {
				lastIncident = &IncidentStatusInfo{
					Date:     component.LastIncident.Date,
					Duration: component.LastIncident.Duration,
					Title:    component.LastIncident.Title,
				}
			}

			result = append(result, ComponentWithSubStatusInfo{
				Component:     component.Component,
				SubComponents: component.SubComponents,
				UptimeHistory: uptimeHistory,
				LastIncident:  lastIncident,
			})
		}

		c.JSON(http.StatusOK, result)
	}
}

func GetStatusIncidents(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		startDate, endDate, err := parseDateRangeParams(c.Query("start_date"), c.Query("end_date"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if startDate == nil && endDate == nil {
			defaultStartDate := time.Now().AddDate(0, 0, -7)
			startDate = &defaultStartDate
		}

		now := time.Now()
		service := statusservice.NewService(repository.NewMongoStatusRepository(db))
		incidents, err := service.BuildIncidents(ctx, now, startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"active":   incidents.Active,
			"resolved": incidents.Resolved,
		})
	}
}

func GetStatusCategory(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		prefix := c.Param("prefix")
		if prefix == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category prefix is required"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		service := statusservice.NewService(repository.NewMongoStatusRepository(db))
		summary, err := service.BuildCategorySummary(ctx, prefix)
		if err != nil {
			if err == statusservice.ErrCategoryNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, summary)
	}
}
