package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	statusservice "github.com/fresp/StatusForge/internal/services/status"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func statusRank(status models.ComponentStatus) int {
	switch status {
	case models.StatusMajorOutage:
		return 5
	case models.StatusPartialOutage:
		return 4
	case models.StatusDegradedPerf:
		return 3
	case models.StatusMaintenance:
		return 2
	case models.StatusOperational:
		return 1
	default:
		return 0
	}
}

func maxStatus(a, b models.ComponentStatus) models.ComponentStatus {
	if statusRank(a) >= statusRank(b) {
		return a
	}
	return b
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

func deriveStatusesFromActiveIncidentsAndMaintenance(
	ctx context.Context,
	db *mongo.Database,
	components []models.Component,
	subs []models.SubComponent,
) (derivedStatusResult, error) {
	activeIncidentsCursor, err := db.Collection("incidents").Find(
		ctx,
		bson.M{"status": bson.M{"$ne": models.IncidentResolved}},
	)
	if err != nil {
		return derivedStatusResult{}, err
	}
	defer activeIncidentsCursor.Close(ctx)

	var activeIncidents []models.Incident
	if err := activeIncidentsCursor.All(ctx, &activeIncidents); err != nil {
		return derivedStatusResult{}, err
	}

	now := time.Now()
	maintenanceCursor, err := db.Collection("maintenance").Find(
		ctx,
		bson.M{
			"status":    models.MaintenanceInProgress,
			"startTime": bson.M{"$lte": now},
			"endTime":   bson.M{"$gte": now},
		},
	)
	if err != nil {
		return derivedStatusResult{}, err
	}
	defer maintenanceCursor.Close(ctx)

	var activeMaintenance []models.Maintenance
	if err := maintenanceCursor.All(ctx, &activeMaintenance); err != nil {
		return derivedStatusResult{}, err
	}

	return deriveStatuses(
		components,
		subs,
		activeIncidents,
		activeMaintenance,
	)
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

func dedupeObjectIDs(ids []primitive.ObjectID) []primitive.ObjectID {
	seen := make(map[primitive.ObjectID]struct{}, len(ids))
	result := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
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

// fetchIncidentUpdates fetches incident updates for multiple incident IDs in batch
func fetchIncidentUpdates(ctx context.Context, db *mongo.Database, incidentIDs []primitive.ObjectID) (map[primitive.ObjectID][]models.IncidentUpdate, error) {
	if len(incidentIDs) == 0 {
		return map[primitive.ObjectID][]models.IncidentUpdate{}, nil
	}

	cursor, err := db.Collection("incident_updates").Find(ctx,
		bson.M{"incidentId": bson.M{"$in": incidentIDs}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	updatesByIncident := map[primitive.ObjectID][]models.IncidentUpdate{}
	var updates []models.IncidentUpdate
	if err := cursor.All(ctx, &updates); err != nil {
		return nil, err
	}

	for _, update := range updates {
		updatesByIncident[update.IncidentID] = append(updatesByIncident[update.IncidentID], update)
	}

	return updatesByIncident, nil
}
func GetStatusSummary(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Get all components
		compCursor, err := db.Collection("components").Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var components []models.Component
		compCursor.All(ctx, &components)
		compCursor.Close(ctx)

		subCursor, err := db.Collection("subcomponents").Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer subCursor.Close(ctx)

		var subComponents []models.SubComponent
		if err := subCursor.All(ctx, &subComponents); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		derivedStatuses, err := deriveStatusesFromActiveIncidentsAndMaintenance(ctx, db, components, subComponents)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		counts := map[string]int{
			"operational":          0,
			"degraded_performance": 0,
			"partial_outage":       0,
			"major_outage":         0,
			"maintenance":          0,
		}
		for _, comp := range components {
			status := derivedStatuses.ComponentStatus[comp.ID]
			counts[string(status)]++
		}

		// Determine overall status
		overall := "operational"
		if counts["major_outage"] > 0 {
			overall = "major_outage"
		} else if counts["partial_outage"] > 0 {
			overall = "partial_outage"
		} else if counts["degraded_performance"] > 0 {
			overall = "degraded_performance"
		} else if counts["maintenance"] > 0 {
			overall = "maintenance"
		}

		// Active incidents
		activeCount, _ := db.Collection("incidents").CountDocuments(ctx,
			bson.M{"status": bson.M{"$ne": models.IncidentResolved}})

		// Scheduled maintenance
		now := time.Now()
		maintCount, _ := db.Collection("maintenance").CountDocuments(ctx,
			bson.M{
				"status":    models.MaintenanceInProgress,
				"startTime": bson.M{"$lte": now},
				"endTime":   bson.M{"$gte": now},
			})

		c.JSON(http.StatusOK, StatusSummary{
			OverallStatus:   overall,
			ComponentCounts: counts,
			ActiveIncidents: int(activeCount),
			ScheduledMaint:  int(maintCount),
		})
	}
}

func GetStatusComponents(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Get components
		compCursor, err := db.Collection("components").Find(ctx, bson.M{},
			options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var components []models.Component
		compCursor.All(ctx, &components)
		compCursor.Close(ctx)

		// Get all subcomponents
		subCursor, _ := db.Collection("subcomponents").Find(ctx, bson.M{})
		var allSubs []models.SubComponent
		if subCursor != nil {
			subCursor.All(ctx, &allSubs)
			subCursor.Close(ctx)
		}

		// Build component map for sub lookup
		subsByComp := map[primitive.ObjectID][]models.SubComponent{}
		for _, s := range allSubs {
			subsByComp[s.ComponentID] = append(subsByComp[s.ComponentID], s)
		}

		// Get monitors for uptime - also considering monitors associated with subcomponents
		monitorCursor, _ := db.Collection("monitors").Find(ctx, bson.M{})
		var monitors []models.Monitor
		if monitorCursor != nil {
			monitorCursor.All(ctx, &monitors)
			monitorCursor.Close(ctx)
		}

		// Get 90-day uptime data
		since := time.Now().AddDate(0, 0, -90)
		uptimeCursor, _ := db.Collection("daily_uptime").Find(ctx,
			bson.M{"date": bson.M{"$gte": since}})
		var uptimeRecords []models.DailyUptime
		if uptimeCursor != nil {
			uptimeCursor.All(ctx, &uptimeRecords)
			uptimeCursor.Close(ctx)
		}

		// Map monitors by component (both direct components and via subcomponents)
		monitorsByComp := map[primitive.ObjectID][]primitive.ObjectID{}
		for _, m := range monitors {
			// Associate with component directly
			if !m.ComponentID.IsZero() {
				monitorsByComp[m.ComponentID] = append(monitorsByComp[m.ComponentID], m.ID)
			}
			// Or associate with component via subcomponent
			if !m.SubComponentID.IsZero() {
				// Find the parent component of this subcomponent
				var subComp models.SubComponent
				err = db.Collection("subcomponents").FindOne(ctx, bson.M{"_id": m.SubComponentID}).Decode(&subComp)
				if err == nil {
					monitorsByComp[subComp.ComponentID] = append(monitorsByComp[subComp.ComponentID], m.ID)
				}
			}
		}

		// Map uptime by monitor
		uptimeByMonitor := map[primitive.ObjectID][]models.DailyUptime{}
		for _, u := range uptimeRecords {
			uptimeByMonitor[u.MonitorID] = append(uptimeByMonitor[u.MonitorID], u)
		}

		derivedStatuses, err := deriveStatusesFromActiveIncidentsAndMaintenance(ctx, db, components, allSubs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for i := range components {
			if status, ok := derivedStatuses.ComponentStatus[components[i].ID]; ok {
				components[i].Status = status
			}
		}

		for i := range allSubs {
			if status, ok := derivedStatuses.SubStatus[allSubs[i].ID]; ok {
				allSubs[i].Status = status
			}
		}

		subsByComp = map[primitive.ObjectID][]models.SubComponent{}
		for _, s := range allSubs {
			subsByComp[s.ComponentID] = append(subsByComp[s.ComponentID], s)
		}

		// Build response with additional outage info
		var result []ComponentWithSubStatusInfo
		for _, comp := range components {
			subs := subsByComp[comp.ID]
			if subs == nil {
				subs = []models.SubComponent{}
			}

			// Build 90-day uptime bars
			bars := build90DayBars(comp.ID, monitorsByComp, uptimeByMonitor)

			// Get incident information for this component
			lastIncidentInfo := getLastIncidentForComponent(ctx, db, comp.ID)

			result = append(result, ComponentWithSubStatusInfo{
				Component:     comp,
				SubComponents: subs,
				UptimeHistory: bars,
				LastIncident:  lastIncidentInfo,
			})
		}

		if result == nil {
			result = []ComponentWithSubStatusInfo{}
		}
		c.JSON(http.StatusOK, result)
	}
}

func build90DayBars(
	compID primitive.ObjectID,
	monitorsByComp map[primitive.ObjectID][]primitive.ObjectID,
	uptimeByMonitor map[primitive.ObjectID][]models.DailyUptime,
) []UptimeBar {

	monitorIDs := monitorsByComp[compID]

	if len(monitorIDs) == 0 {
		return []UptimeBar{}
	}

	bars := make([]UptimeBar, 90)
	now := time.Now()

	for i := 89; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayStr := day.Format("2006-01-02")

		bars[89-i] = UptimeBar{
			Date:          dayStr,
			UptimePercent: 100.0,
			Status:        "operational",
		}

		var totalUp, total int

		for _, mID := range monitorIDs {
			for _, u := range uptimeByMonitor[mID] {
				if u.Date.Format("2006-01-02") == dayStr {
					totalUp += u.SuccessfulChecks
					total += u.TotalChecks
				}
			}
		}

		if total > 0 {
			pct := float64(totalUp) / float64(total) * 100.0
			bars[89-i].UptimePercent = pct

			switch {
			case pct >= 99.9:
				bars[89-i].Status = "operational"
			case pct >= 90.0:
				bars[89-i].Status = "degraded_performance"
			case pct >= 50.0:
				bars[89-i].Status = "partial_outage"
			default:
				bars[89-i].Status = "major_outage"
			}
		}
	}

	return bars
}

// getLastIncidentForComponent retrieves the most recent incident associated with a component
func getLastIncidentForComponent(ctx context.Context, db *mongo.Database, compID primitive.ObjectID) *IncidentStatusInfo {
	inFilter := bson.M{
		"$or": []bson.M{
			{"affectedComponents": bson.M{"$in": []primitive.ObjectID{compID}}},
			{"affectedComponentTargets.componentId": compID},
		},
	}

	var incident models.Incident
	err := db.Collection("incidents").FindOne(ctx,
		inFilter,
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&incident)

	// If no incident was found, return nil
	if err != nil {
		return nil
	}

	// Calculate duration if incident was resolved
	durationStr := "Unknown"
	if incident.ResolvedAt != nil && !incident.ResolvedAt.IsZero() {
		dur := incident.ResolvedAt.Sub(incident.CreatedAt)
		// Simplify duration for display
		if dur.Minutes() < 60 {
			durationStr = fmt.Sprintf("%.0f minute(s)", dur.Minutes())
		} else if dur.Hours() < 24 {
			durationStr = fmt.Sprintf("%.1f hour(s)", dur.Hours())
		} else {
			days := dur.Hours() / 24
			durationStr = fmt.Sprintf("%.1f day(s)", days)
		}
	} else {
		// For unresolved incidents, calculate from start to now
		dur := time.Since(incident.CreatedAt)
		if dur.Minutes() < 60 {
			durationStr = fmt.Sprintf("%.0f minute(s)", dur.Minutes())
		} else if dur.Hours() < 24 {
			durationStr = fmt.Sprintf("%.1f hour(s)", dur.Hours())
		} else {
			days := dur.Hours() / 24
			durationStr = fmt.Sprintf("%.1f day(s)", days)
		}
	}

	return &IncidentStatusInfo{
		Date:     incident.CreatedAt,
		Duration: durationStr,
		Title:    incident.Title,
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

		var incidents []models.Incident

		// =========================
		// FETCH INCIDENTS (UNIFIED)
		// =========================
		if startDate != nil || endDate != nil {
			dateFilter := bson.M{}
			if startDate != nil {
				dateFilter["$gte"] = *startDate
			}
			if endDate != nil {
				dateFilter["$lt"] = *endDate
			}

			cursor, err := db.Collection("incidents").Find(ctx,
				bson.M{"createdAt": dateFilter},
				options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer cursor.Close(ctx)

			cursor.All(ctx, &incidents)

		} else {
			// Active incidents
			activeCursor, err := db.Collection("incidents").Find(ctx,
				bson.M{"status": bson.M{"$ne": models.IncidentResolved}},
				options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer activeCursor.Close(ctx)

			var activeIncidents []models.Incident
			activeCursor.All(ctx, &activeIncidents)

			// Resolved (last 30 days)
			since30 := time.Now().AddDate(0, 0, -30)
			resolvedCursor, _ := db.Collection("incidents").Find(ctx,
				bson.M{
					"status":     models.IncidentResolved,
					"resolvedAt": bson.M{"$gte": since30},
				},
				options.Find().SetSort(bson.D{{Key: "resolvedAt", Value: -1}}),
			)

			var resolvedIncidents []models.Incident
			if resolvedCursor != nil {
				defer resolvedCursor.Close(ctx)
				resolvedCursor.All(ctx, &resolvedIncidents)
			}

			incidents = append(activeIncidents, resolvedIncidents...)
		}

		if incidents == nil {
			incidents = []models.Incident{}
		}

		// =========================
		// FETCH UPDATES (BATCH)
		// =========================
		allIDs := make([]primitive.ObjectID, 0, len(incidents))
		for _, inc := range incidents {
			allIDs = append(allIDs, inc.ID)
		}

		updatesMap, err := fetchIncidentUpdates(ctx, db, allIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// =========================
		// 🔥 FETCH COMPONENTS (NEW)
		// =========================
		componentIDSet := make(map[primitive.ObjectID]struct{})
		subComponentIDSet := make(map[primitive.ObjectID]struct{})
		for _, inc := range incidents {
			for _, compID := range inc.AffectedComponents {
				componentIDSet[compID] = struct{}{}
			}
			for _, target := range normalizeIncidentTargetsForExpansion(inc) {
				componentIDSet[target.ComponentID] = struct{}{}
				for _, subID := range target.SubComponentIDs {
					subComponentIDSet[subID] = struct{}{}
				}
			}
		}

		componentIDs := make([]primitive.ObjectID, 0, len(componentIDSet))
		for id := range componentIDSet {
			componentIDs = append(componentIDs, id)
		}

		componentMap := make(map[primitive.ObjectID]models.Component)
		subComponentMap := make(map[primitive.ObjectID]models.SubComponent)

		if len(componentIDs) > 0 {
			cursor, err := db.Collection("components").Find(ctx, bson.M{
				"_id": bson.M{"$in": componentIDs},
			})
			if err == nil {
				defer cursor.Close(ctx)

				var components []models.Component
				cursor.All(ctx, &components)

				for _, comp := range components {
					componentMap[comp.ID] = comp
				}
			}
		}

		subComponentIDs := make([]primitive.ObjectID, 0, len(subComponentIDSet))
		for id := range subComponentIDSet {
			subComponentIDs = append(subComponentIDs, id)
		}

		if len(subComponentIDs) > 0 {
			cursor, err := db.Collection("subcomponents").Find(ctx, bson.M{
				"_id": bson.M{"$in": subComponentIDs},
			})
			if err == nil {
				defer cursor.Close(ctx)

				var subComponents []models.SubComponent
				cursor.All(ctx, &subComponents)

				for _, subComponent := range subComponents {
					subComponentMap[subComponent.ID] = subComponent
				}
			}
		}

		// =========================
		// BUILD RESPONSE
		// =========================
		activeWithUpdates := []models.IncidentWithUpdates{}
		resolvedWithUpdates := []models.IncidentWithUpdates{}

		for _, inc := range incidents {
			targets := normalizeIncidentTargetsForExpansion(inc)

			componentIDs := make([]primitive.ObjectID, 0, len(inc.AffectedComponents)+len(targets))
			componentIDs = append(componentIDs, inc.AffectedComponents...)
			for _, target := range targets {
				componentIDs = append(componentIDs, target.ComponentID)
			}

			componentIDs = dedupeObjectIDs(componentIDs)

			expandedComponents := make([]models.Component, 0, len(componentIDs))
			for _, compID := range componentIDs {
				if comp, ok := componentMap[compID]; ok {
					expandedComponents = append(expandedComponents, comp)
				}
			}

			expandedTargets := make([]models.IncidentAffectedComponentExpanded, 0, len(targets))
			for _, target := range targets {
				component, ok := componentMap[target.ComponentID]
				if !ok {
					continue
				}

				expandedSubComponents := make([]models.SubComponent, 0, len(target.SubComponentIDs))
				for _, subID := range target.SubComponentIDs {
					if subComponent, exists := subComponentMap[subID]; exists {
						expandedSubComponents = append(expandedSubComponents, subComponent)
					}
				}

				expandedTargets = append(expandedTargets, models.IncidentAffectedComponentExpanded{
					Component:     component,
					SubComponents: expandedSubComponents,
				})
			}

			item := models.IncidentWithUpdates{
				Incident:                 inc,
				Updates:                  updatesMap[inc.ID],
				AffectedComponents:       expandedComponents,
				AffectedComponentTargets: expandedTargets,
			}

			if inc.Status == models.IncidentResolved {
				resolvedWithUpdates = append(resolvedWithUpdates, item)
			} else {
				activeWithUpdates = append(activeWithUpdates, item)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"active":   activeWithUpdates,
			"resolved": resolvedWithUpdates,
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
