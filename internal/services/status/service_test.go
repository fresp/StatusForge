package status

import (
	"context"
	"testing"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubStatusRepo struct {
	components              []models.Component
	componentsByID          map[primitive.ObjectID]models.Component
	subComponents           []models.SubComponent
	subComponentsByID       map[primitive.ObjectID]models.SubComponent
	monitors                []models.Monitor
	monitorsByService       []models.Monitor
	monitorLogs             []models.MonitorLog
	dailyUptime             []models.DailyUptime
	incidentsByAffected     []models.Incident
	incidentUpdates         map[primitive.ObjectID][]models.IncidentUpdate
	activeIncidents         []models.Incident
	activeMaintenance       []models.Maintenance
	activeIncidentCount     int64
	activeMaintenanceCount  int64
	latestIncidentByComp    map[primitive.ObjectID]*models.Incident
	incidentsByCreatedRange []models.Incident
	resolvedSince           []models.Incident
	err                     error
}

func (r *stubStatusRepo) ListComponents(_ context.Context) ([]models.Component, error) {
	return r.components, r.err
}

func (r *stubStatusRepo) ListSubComponentsByComponentIDs(_ context.Context, componentIDs []primitive.ObjectID) ([]models.SubComponent, error) {
	if len(componentIDs) == 0 {
		return []models.SubComponent{}, r.err
	}

	allowed := map[primitive.ObjectID]struct{}{}
	for _, id := range componentIDs {
		allowed[id] = struct{}{}
	}

	result := make([]models.SubComponent, 0)
	for _, sub := range r.subComponents {
		if _, ok := allowed[sub.ComponentID]; ok {
			result = append(result, sub)
		}
	}

	return result, r.err
}

func (r *stubStatusRepo) ListMonitorsByTargets(_ context.Context, componentIDs []primitive.ObjectID, subComponentIDs []primitive.ObjectID) ([]models.Monitor, error) {
	allowedComponents := map[primitive.ObjectID]struct{}{}
	allowedSubs := map[primitive.ObjectID]struct{}{}
	for _, id := range componentIDs {
		allowedComponents[id] = struct{}{}
	}
	for _, id := range subComponentIDs {
		allowedSubs[id] = struct{}{}
	}

	result := make([]models.Monitor, 0)
	for _, monitor := range r.monitors {
		if _, ok := allowedComponents[monitor.ComponentID]; ok {
			result = append(result, monitor)
			continue
		}
		if _, ok := allowedSubs[monitor.SubComponentID]; ok {
			result = append(result, monitor)
		}
	}

	return result, r.err
}

func (r *stubStatusRepo) ListMonitorsByServiceID(_ context.Context, _ primitive.ObjectID) ([]models.Monitor, error) {
	return r.monitorsByService, r.err
}

func (r *stubStatusRepo) FindMonitorByID(_ context.Context, id primitive.ObjectID) (*models.Monitor, error) {
	for _, monitor := range r.monitorsByService {
		if monitor.ID == id {
			m := monitor
			return &m, r.err
		}
	}
	for _, monitor := range r.monitors {
		if monitor.ID == id {
			m := monitor
			return &m, r.err
		}
	}
	return nil, r.err
}

func (r *stubStatusRepo) FindMonitorBySubComponentID(_ context.Context, subComponentID primitive.ObjectID) (*models.Monitor, error) {
	for _, monitor := range r.monitorsByService {
		if monitor.SubComponentID == subComponentID {
			m := monitor
			return &m, r.err
		}
	}

	for _, monitor := range r.monitors {
		if monitor.SubComponentID == subComponentID {
			m := monitor
			return &m, r.err
		}
	}

	return nil, r.err
}

func (r *stubStatusRepo) ListMonitorLogsByMonitorIDsSince(_ context.Context, monitorIDs []primitive.ObjectID, since time.Time) ([]models.MonitorLog, error) {
	allowed := map[primitive.ObjectID]struct{}{}
	for _, id := range monitorIDs {
		allowed[id] = struct{}{}
	}

	result := make([]models.MonitorLog, 0)
	for _, log := range r.monitorLogs {
		if _, ok := allowed[log.MonitorID]; !ok {
			continue
		}
		if log.CheckedAt.Before(since) {
			continue
		}
		result = append(result, log)
	}

	return result, r.err
}

func (r *stubStatusRepo) ListDailyUptimeSinceByMonitorIDs(_ context.Context, monitorIDs []primitive.ObjectID, since time.Time) ([]models.DailyUptime, error) {
	allowed := map[primitive.ObjectID]struct{}{}
	for _, id := range monitorIDs {
		allowed[id] = struct{}{}
	}

	result := make([]models.DailyUptime, 0)
	for _, record := range r.dailyUptime {
		if _, ok := allowed[record.MonitorID]; !ok {
			continue
		}
		if record.Date.Before(since) {
			continue
		}
		result = append(result, record)
	}

	return result, r.err
}

func (r *stubStatusRepo) ListIncidentsByAffectedComponents(_ context.Context, affectedIDs []primitive.ObjectID, _ int64) ([]models.Incident, error) {
	return r.incidentsByAffected, r.err
}

func (r *stubStatusRepo) ListIncidentUpdatesByIncidentIDs(_ context.Context, incidentIDs []primitive.ObjectID) (map[primitive.ObjectID][]models.IncidentUpdate, error) {
	result := map[primitive.ObjectID][]models.IncidentUpdate{}
	for _, id := range incidentIDs {
		result[id] = append([]models.IncidentUpdate(nil), r.incidentUpdates[id]...)
	}
	return result, r.err
}

func (r *stubStatusRepo) ListAllSubComponents(_ context.Context) ([]models.SubComponent, error) {
	return r.subComponents, r.err
}

func (r *stubStatusRepo) ListActiveIncidents(_ context.Context) ([]models.Incident, error) {
	return r.activeIncidents, r.err
}

func (r *stubStatusRepo) ListActiveMaintenanceAt(_ context.Context, _ time.Time) ([]models.Maintenance, error) {
	return r.activeMaintenance, r.err
}

func (r *stubStatusRepo) CountActiveIncidents(_ context.Context) (int64, error) {
	return r.activeIncidentCount, r.err
}

func (r *stubStatusRepo) CountActiveMaintenanceAt(_ context.Context, _ time.Time) (int64, error) {
	return r.activeMaintenanceCount, r.err
}

func (r *stubStatusRepo) FindLatestIncidentByComponent(_ context.Context, componentID primitive.ObjectID) (*models.Incident, error) {
	if incident := r.latestIncidentByComp[componentID]; incident != nil {
		copy := *incident
		return &copy, r.err
	}
	return nil, r.err
}

func (r *stubStatusRepo) ListIncidentsByCreatedAtRange(_ context.Context, _, _ time.Time) ([]models.Incident, error) {
	return r.incidentsByCreatedRange, r.err
}

func (r *stubStatusRepo) ListResolvedIncidentsSince(_ context.Context, _ time.Time) ([]models.Incident, error) {
	return r.resolvedSince, r.err
}

func (r *stubStatusRepo) ListComponentsByIDs(_ context.Context, ids []primitive.ObjectID) ([]models.Component, error) {
	result := make([]models.Component, 0, len(ids))
	for _, id := range ids {
		if component, ok := r.componentsByID[id]; ok {
			result = append(result, component)
		}
	}
	return result, r.err
}

func (r *stubStatusRepo) ListSubComponentsByIDs(_ context.Context, ids []primitive.ObjectID) ([]models.SubComponent, error) {
	result := make([]models.SubComponent, 0, len(ids))
	for _, id := range ids {
		if subComponent, ok := r.subComponentsByID[id]; ok {
			result = append(result, subComponent)
		}
	}
	return result, r.err
}

func TestBuildSummaryDerivesOverallStatusAndCounts(t *testing.T) {
	componentA := primitive.NewObjectID()
	componentB := primitive.NewObjectID()
	subA := primitive.NewObjectID()
	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)

	repo := &stubStatusRepo{
		components: []models.Component{
			{ID: componentA, Name: "API", Status: models.StatusOperational},
			{ID: componentB, Name: "Web", Status: models.StatusOperational},
		},
		subComponents: []models.SubComponent{
			{ID: subA, ComponentID: componentA, Name: "API worker", Status: models.StatusOperational},
		},
		activeIncidents: []models.Incident{{
			Impact: models.ImpactCritical,
			AffectedComponentTargets: []models.IncidentAffectedComponent{{
				ComponentID:     componentA,
				SubComponentIDs: []primitive.ObjectID{subA},
			}},
		}},
		activeMaintenance:      []models.Maintenance{{Components: []primitive.ObjectID{componentB}}},
		activeIncidentCount:    3,
		activeMaintenanceCount: 2,
	}

	svc := NewService(repo)
	summary, err := svc.BuildSummary(context.Background(), now)
	require.NoError(t, err)

	assert.Equal(t, "major_outage", summary.OverallStatus)
	assert.Equal(t, 0, summary.ComponentCounts["operational"])
	assert.Equal(t, 0, summary.ComponentCounts["degraded_performance"])
	assert.Equal(t, 0, summary.ComponentCounts["partial_outage"])
	assert.Equal(t, 1, summary.ComponentCounts["major_outage"])
	assert.Equal(t, 1, summary.ComponentCounts["maintenance"])
	assert.Equal(t, 3, summary.ActiveIncidents)
	assert.Equal(t, 2, summary.ScheduledMaintenance)
}

func TestBuildComponentsExpandsSubcomponentsUptimeAndLastIncident(t *testing.T) {
	componentID := primitive.NewObjectID()
	subID := primitive.NewObjectID()
	directMonitorID := primitive.NewObjectID()
	subMonitorID := primitive.NewObjectID()
	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	resolvedAt := now.Add(-2 * time.Hour)
	createdAt := now.Add(-5 * time.Hour)

	repo := &stubStatusRepo{
		components: []models.Component{{
			ID:     componentID,
			Name:   "Checkout",
			Status: models.StatusOperational,
		}},
		subComponents: []models.SubComponent{{
			ID:          subID,
			ComponentID: componentID,
			Name:        "Worker",
			Status:      models.StatusOperational,
		}},
		monitors: []models.Monitor{
			{ID: directMonitorID, ComponentID: componentID},
			{ID: subMonitorID, SubComponentID: subID},
		},
		dailyUptime: []models.DailyUptime{
			{MonitorID: directMonitorID, Date: now.AddDate(0, 0, -1), UptimePercent: 99.9},
			{MonitorID: subMonitorID, Date: now.AddDate(0, 0, -1), UptimePercent: 50},
		},
		activeIncidents: []models.Incident{{
			Impact: models.ImpactCritical,
			AffectedComponentTargets: []models.IncidentAffectedComponent{{
				ComponentID:     componentID,
				SubComponentIDs: []primitive.ObjectID{subID},
			}},
		}},
		latestIncidentByComp: map[primitive.ObjectID]*models.Incident{
			componentID: {
				ID:         primitive.NewObjectID(),
				Title:      "Checkout outage",
				CreatedAt:  createdAt,
				ResolvedAt: &resolvedAt,
			},
		},
	}

	svc := NewService(repo)
	components, err := svc.BuildComponents(context.Background(), now)
	require.NoError(t, err)
	require.Len(t, components, 1)

	component := components[0]
	assert.Equal(t, models.StatusPartialOutage, component.Status)
	require.Len(t, component.SubComponents, 1)
	assert.Equal(t, models.StatusMajorOutage, component.SubComponents[0].Status)
	assert.NotEmpty(t, component.UptimeHistory)
	require.NotNil(t, component.LastIncident)
	assert.Equal(t, "Checkout outage", component.LastIncident.Title)
	assert.Equal(t, createdAt, component.LastIncident.Date)
	assert.Equal(t, "3.0 hour(s)", component.LastIncident.Duration)
}

func TestBuildIncidentsDefaultsToRecentWindowAndExpandsTargets(t *testing.T) {
	componentID := primitive.NewObjectID()
	subID := primitive.NewObjectID()
	activeIncidentID := primitive.NewObjectID()
	resolvedRecentID := primitive.NewObjectID()
	resolvedOldID := primitive.NewObjectID()
	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)

	activeIncident := models.Incident{
		ID:        activeIncidentID,
		Title:     "API degraded",
		Status:    models.IncidentInvestigating,
		Impact:    models.ImpactMajor,
		CreatedAt: now.Add(-2 * time.Hour),
		AffectedComponentTargets: []models.IncidentAffectedComponent{{
			ComponentID:     componentID,
			SubComponentIDs: []primitive.ObjectID{subID},
		}},
	}
	resolvedRecent := models.Incident{
		ID:                 resolvedRecentID,
		Title:              "API resolved",
		Status:             models.IncidentResolved,
		Impact:             models.ImpactMinor,
		CreatedAt:          now.AddDate(0, 0, -2),
		AffectedComponents: []primitive.ObjectID{componentID},
	}
	resolvedOld := models.Incident{
		ID:                 resolvedOldID,
		Title:              "Old issue",
		Status:             models.IncidentResolved,
		Impact:             models.ImpactMinor,
		CreatedAt:          now.AddDate(0, 0, -45),
		AffectedComponents: []primitive.ObjectID{componentID},
	}

	repo := &stubStatusRepo{
		activeIncidents: []models.Incident{activeIncident},
		resolvedSince:   []models.Incident{resolvedRecent},
		componentsByID: map[primitive.ObjectID]models.Component{
			componentID: {ID: componentID, Name: "API", Status: models.StatusOperational},
		},
		subComponentsByID: map[primitive.ObjectID]models.SubComponent{
			subID: {ID: subID, ComponentID: componentID, Name: "Worker", Status: models.StatusOperational},
		},
		incidentUpdates: map[primitive.ObjectID][]models.IncidentUpdate{
			activeIncidentID: {{IncidentID: activeIncidentID, Message: "Investigating", Status: models.IncidentInvestigating, CreatedAt: now.Add(-90 * time.Minute)}},
		},
	}

	svc := NewService(repo)
	response, err := svc.BuildIncidents(context.Background(), now, nil, nil)
	require.NoError(t, err)

	require.Len(t, response.Active, 1)
	require.Len(t, response.Resolved, 1)
	assert.Equal(t, activeIncidentID, response.Active[0].ID)
	assert.Equal(t, resolvedRecentID, response.Resolved[0].ID)
	assert.NotEqual(t, resolvedOld.ID, response.Resolved[0].ID)
	require.Len(t, response.Active[0].AffectedComponents, 1)
	assert.Equal(t, "API", response.Active[0].AffectedComponents[0].Name)
	require.Len(t, response.Active[0].AffectedComponentTargets, 1)
	require.Len(t, response.Active[0].AffectedComponentTargets[0].SubComponents, 1)
	assert.Equal(t, "Worker", response.Active[0].AffectedComponentTargets[0].SubComponents[0].Name)
	require.Len(t, response.Active[0].Updates, 1)
}

func TestBuildServiceMetricsAggregatesLatencyAvailabilityAndMonthlyHistory(t *testing.T) {
	serviceID := primitive.NewObjectID()
	monitorID := primitive.NewObjectID()
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	repo := &stubStatusRepo{
		monitorsByService: []models.Monitor{{ID: monitorID}},
		monitorLogs: []models.MonitorLog{
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 100, CheckedAt: time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 200, CheckedAt: time.Date(2026, 1, 20, 8, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorDown, ResponseTime: 300, CheckedAt: time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 400, CheckedAt: time.Date(2026, 2, 10, 8, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 500, CheckedAt: time.Date(2026, 2, 20, 8, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 100, CheckedAt: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 200, CheckedAt: time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorDown, ResponseTime: 300, CheckedAt: time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 400, CheckedAt: time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 500, CheckedAt: time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 600, CheckedAt: time.Date(2026, 3, 6, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 700, CheckedAt: time.Date(2026, 3, 7, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorDown, ResponseTime: 800, CheckedAt: time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 900, CheckedAt: time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 1000, CheckedAt: time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)},
		},
	}

	svc := NewService(repo)
	metrics, err := svc.BuildServiceMetrics(context.Background(), serviceID, now)
	require.NoError(t, err)
	require.NotNil(t, metrics)
	require.NotNil(t, metrics.Latency)
	require.NotNil(t, metrics.Availability)

	assert.Equal(t, float64(900), metrics.Latency.P90)
	assert.Equal(t, float64(1000), metrics.Latency.P99)
	assert.InDelta(t, 77.777, metrics.Availability.Last30Days, 0.001)

	require.NotEmpty(t, metrics.History)
	assert.Equal(t, "January 2026", metrics.History[0].Month)
	assert.Equal(t, float64(200), metrics.History[0].Latency.P90)
	assert.Equal(t, float64(200), metrics.History[0].Latency.P99)
	assert.InDelta(t, 100.0, metrics.History[0].Availability, 0.001)

	assert.Equal(t, "February 2026", metrics.History[1].Month)
	assert.Equal(t, float64(500), metrics.History[1].Latency.P90)
	assert.Equal(t, float64(500), metrics.History[1].Latency.P99)
	assert.InDelta(t, 66.666, metrics.History[1].Availability, 0.01)

	assert.Equal(t, "March 2026", metrics.History[2].Month)
	assert.Equal(t, float64(900), metrics.History[2].Latency.P90)
	assert.Equal(t, float64(1000), metrics.History[2].Latency.P99)
	assert.InDelta(t, 80.0, metrics.History[2].Availability, 0.001)
}

func TestBuildServiceMetricsReturnsEmptyMetricsWhenServiceHasNoMonitors(t *testing.T) {
	svc := NewService(&stubStatusRepo{monitorsByService: []models.Monitor{}})
	metrics, err := svc.BuildServiceMetrics(context.Background(), primitive.NewObjectID(), time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.Nil(t, metrics.Latency)
	assert.Nil(t, metrics.Availability)
	assert.Empty(t, metrics.History)
}

func TestBuildServiceMetricsHistoryContainsOnlyLastThreeMonths(t *testing.T) {
	serviceID := primitive.NewObjectID()
	monitorID := primitive.NewObjectID()
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	repo := &stubStatusRepo{
		monitorsByService: []models.Monitor{{ID: monitorID}},
		monitorLogs: []models.MonitorLog{
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 100, CheckedAt: time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 200, CheckedAt: time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 300, CheckedAt: time.Date(2026, 2, 15, 9, 0, 0, 0, time.UTC)},
			{MonitorID: monitorID, Status: models.MonitorUp, ResponseTime: 400, CheckedAt: time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)},
		},
	}

	svc := NewService(repo)
	metrics, err := svc.BuildServiceMetrics(context.Background(), serviceID, now)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	require.Len(t, metrics.History, 3)
	assert.Equal(t, "January 2026", metrics.History[0].Month)
	assert.Equal(t, "February 2026", metrics.History[1].Month)
	assert.Equal(t, "March 2026", metrics.History[2].Month)
}
