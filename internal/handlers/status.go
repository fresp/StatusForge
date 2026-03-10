package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"status-platform/internal/models"
)

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

		counts := map[string]int{
			"operational":          0,
			"degraded_performance": 0,
			"partial_outage":       0,
			"major_outage":         0,
			"maintenance":          0,
		}
		for _, comp := range components {
			counts[string(comp.Status)]++
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
		maintCount, _ := db.Collection("maintenance").CountDocuments(ctx,
			bson.M{"status": bson.M{"$in": []string{"scheduled", "in_progress"}}})

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

func build90DayBars(compID primitive.ObjectID, monitorsByComp map[primitive.ObjectID][]primitive.ObjectID, uptimeByMonitor map[primitive.ObjectID][]models.DailyUptime) []UptimeBar {
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

		// Find monitor data for this component on this day
		monitorIDs := monitorsByComp[compID]
		if len(monitorIDs) == 0 {
			continue
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
			if pct >= 99.9 {
				bars[89-i].Status = "operational"
			} else if pct >= 90.0 {
				bars[89-i].Status = "degraded_performance"
			} else if pct >= 50.0 {
				bars[89-i].Status = "partial_outage"
			} else {
				bars[89-i].Status = "major_outage"
			}
		}
	}
	return bars
}

// getLastIncidentForComponent retrieves the most recent incident associated with a component
func getLastIncidentForComponent(ctx context.Context, db *mongo.Database, compID primitive.ObjectID) *IncidentStatusInfo {
	inFilter := bson.M{"affectedComponents": bson.M{"$in": []primitive.ObjectID{compID}}}

	var incident models.Incident
	err := db.Collection("incidents").FindOne(ctx,
		inFilter,
		options.FindOne().SetSort(bson.D{{"createdAt", -1}})).Decode(&incident)

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

		// Active incidents
		activeCursor, err := db.Collection("incidents").Find(ctx,
			bson.M{"status": bson.M{"$ne": models.IncidentResolved}},
			options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var activeIncidents []models.Incident
		activeCursor.All(ctx, &activeIncidents)
		activeCursor.Close(ctx)

		// Past 30 days resolved
		since30 := time.Now().AddDate(0, 0, -30)
		resolvedCursor, _ := db.Collection("incidents").Find(ctx,
			bson.M{"status": models.IncidentResolved, "resolvedAt": bson.M{"$gte": since30}},
			options.Find().SetSort(bson.D{{Key: "resolvedAt", Value: -1}}))
		var resolvedIncidents []models.Incident
		if resolvedCursor != nil {
			resolvedCursor.All(ctx, &resolvedIncidents)
			resolvedCursor.Close(ctx)
		}
		if activeIncidents == nil {
			activeIncidents = []models.Incident{}
		}
		if resolvedIncidents == nil {
			resolvedIncidents = []models.Incident{}
		}

		// Collect all incident IDs for batch update fetch
		allIDs := make([]primitive.ObjectID, 0, len(activeIncidents)+len(resolvedIncidents))
		for _, inc := range activeIncidents {
			allIDs = append(allIDs, inc.ID)
		}
		for _, inc := range resolvedIncidents {
			allIDs = append(allIDs, inc.ID)
		}

		// Fetch all updates in a single batch query
		updatesMap, err := fetchIncidentUpdates(ctx, db, allIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Attach updates to each incident
		var activeWithUpdates []models.IncidentWithUpdates
		for _, inc := range activeIncidents {
			activeWithUpdates = append(activeWithUpdates, models.IncidentWithUpdates{
				Incident: inc,
				Updates:  updatesMap[inc.ID],
			})
		}

		var resolvedWithUpdates []models.IncidentWithUpdates
		for _, inc := range resolvedIncidents {
			resolvedWithUpdates = append(resolvedWithUpdates, models.IncidentWithUpdates{
				Incident: inc,
				Updates:  updatesMap[inc.ID],
			})
		}

		if activeWithUpdates == nil {
			activeWithUpdates = []models.IncidentWithUpdates{}
		}
		if resolvedWithUpdates == nil {
			resolvedWithUpdates = []models.IncidentWithUpdates{}
		}

		c.JSON(http.StatusOK, gin.H{
			"active":   activeWithUpdates,
			"resolved": resolvedWithUpdates,
		})
	}
}
