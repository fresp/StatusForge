package status

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	idsdomain "github.com/fresp/StatusForge/internal/domain/ids"
	statusdomain "github.com/fresp/StatusForge/internal/domain/status"
	uptimedomain "github.com/fresp/StatusForge/internal/domain/uptime"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	repo repository.StatusRepository
}

func NewService(repo repository.StatusRepository) *Service {
	return &Service{repo: repo}
}

type CategoryService struct {
	ID            primitive.ObjectID     `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Status        models.ComponentStatus `json:"status"`
	Uptime90d     float64                `json:"uptime90d"`
	UptimeHistory []DailyUptimeBar       `json:"uptimeHistory"`
}

type CategorySummary struct {
	Prefix          string                       `json:"prefix"`
	Name            string                       `json:"name"`
	Description     string                       `json:"description"`
	AggregateStatus string                       `json:"aggregateStatus"`
	Uptime90d       float64                      `json:"uptime90d"`
	Services        []CategoryService            `json:"services"`
	Incidents       []models.IncidentWithUpdates `json:"incidents"`
}

type DailyUptimeBar struct {
	Date          string  `json:"date"`
	UptimePercent float64 `json:"uptimePercent"`
	Status        string  `json:"status"`
}

type StatusSummary struct {
	OverallStatus        string         `json:"overallStatus"`
	ComponentCounts      map[string]int `json:"componentCounts"`
	ActiveIncidents      int            `json:"activeIncidents"`
	ScheduledMaintenance int            `json:"scheduledMaintenance"`
}

type ComponentWithSubStatusInfo struct {
	models.Component
	SubComponents []models.SubComponent `json:"subComponents"`
	UptimeHistory []DailyUptimeBar      `json:"uptimeHistory"`
	LastIncident  *IncidentStatusInfo   `json:"lastIncident,omitempty"`
}

type IncidentStatusInfo struct {
	Date     time.Time `json:"date"`
	Duration string    `json:"duration"`
	Title    string    `json:"title"`
}

type IncidentsResponse struct {
	Active   []models.IncidentWithUpdates `json:"active"`
	Resolved []models.IncidentWithUpdates `json:"resolved"`
}

type derivedStatusResult struct {
	ComponentStatus map[primitive.ObjectID]models.ComponentStatus
	SubStatus       map[primitive.ObjectID]models.ComponentStatus
}

type componentAggregationState struct {
	HasDirectImpact bool
	DirectStatus    models.ComponentStatus
	ImpactedSubIDs  map[primitive.ObjectID]struct{}
}

func (s *Service) BuildCategorySummary(ctx context.Context, prefix string) (*CategorySummary, error) {
	components, err := s.repo.ListComponents(ctx)
	if err != nil {
		return nil, err
	}

	if len(components) == 0 {
		return nil, ErrCategoryNotFound
	}

	categoryComponent := findCategoryComponent(components, prefix)
	if categoryComponent == nil {
		return nil, ErrCategoryNotFound
	}

	subs, err := s.repo.ListSubComponentsByComponentIDs(ctx, []primitive.ObjectID{categoryComponent.ID})
	if err != nil {
		return nil, err
	}

	subIDs := make([]primitive.ObjectID, 0, len(subs))
	for _, sub := range subs {
		subIDs = append(subIDs, sub.ID)
	}

	monitors, err := s.repo.ListMonitorsByTargets(ctx, []primitive.ObjectID{categoryComponent.ID}, subIDs)
	if err != nil {
		return nil, err
	}

	monitorIDs := make([]primitive.ObjectID, 0, len(monitors))
	for _, monitor := range monitors {
		monitorIDs = append(monitorIDs, monitor.ID)
	}

	uptimeRecords, err := s.repo.ListDailyUptimeSinceByMonitorIDs(ctx, monitorIDs, time.Now().AddDate(0, 0, -90))
	if err != nil {
		return nil, err
	}

	monitorsBySubID := map[primitive.ObjectID][]primitive.ObjectID{}
	componentMonitorIDs := []primitive.ObjectID{}
	for _, monitor := range monitors {
		if !monitor.SubComponentID.IsZero() {
			monitorsBySubID[monitor.SubComponentID] = append(monitorsBySubID[monitor.SubComponentID], monitor.ID)
			continue
		}
		if !monitor.ComponentID.IsZero() && monitor.ComponentID == categoryComponent.ID {
			componentMonitorIDs = append(componentMonitorIDs, monitor.ID)
		}
	}

	uptimeByMonitorID := map[primitive.ObjectID][]models.DailyUptime{}
	for _, record := range uptimeRecords {
		uptimeByMonitorID[record.MonitorID] = append(uptimeByMonitorID[record.MonitorID], record)
	}

	services := make([]CategoryService, 0, len(subs))
	if len(subs) > 0 {
		for _, sub := range subs {
			history := build90DayBars(monitorsBySubID[sub.ID], uptimeByMonitorID)
			services = append(services, CategoryService{
				ID:            sub.ID,
				Name:          sub.Name,
				Description:   sub.Description,
				Status:        sub.Status,
				Uptime90d:     averageUptime(history),
				UptimeHistory: history,
			})
		}
	} else {
		history := build90DayBars(componentMonitorIDs, uptimeByMonitorID)
		services = append(services, CategoryService{
			ID:            categoryComponent.ID,
			Name:          categoryComponent.Name,
			Description:   categoryComponent.Description,
			Status:        categoryComponent.Status,
			Uptime90d:     averageUptime(history),
			UptimeHistory: history,
		})
	}

	aggregateStatus := aggregateStatusFromServices(services)
	categoryUptime := 0.0
	if len(services) > 0 {
		total := 0.0
		for _, service := range services {
			total += service.Uptime90d
		}
		categoryUptime = total / float64(len(services))
	}

	affectedTargets := []primitive.ObjectID{categoryComponent.ID}
	affectedTargets = append(affectedTargets, subIDs...)

	incidents, err := s.repo.ListIncidentsByAffectedComponents(ctx, affectedTargets, 20)
	if err != nil {
		return nil, err
	}

	incidentIDs := make([]primitive.ObjectID, 0, len(incidents))
	for _, incident := range incidents {
		incidentIDs = append(incidentIDs, incident.ID)
	}

	updatesByIncident, err := s.repo.ListIncidentUpdatesByIncidentIDs(ctx, incidentIDs)
	if err != nil {
		return nil, err
	}

	incidentsWithUpdates := make([]models.IncidentWithUpdates, 0, len(incidents))

	incidentComponentMap := map[primitive.ObjectID]models.Component{}
	incidentSubComponentMap := map[primitive.ObjectID]models.SubComponent{}

	for _, component := range components {
		incidentComponentMap[component.ID] = component
	}
	for _, subComponent := range subs {
		incidentSubComponentMap[subComponent.ID] = subComponent
	}

	for _, incident := range incidents {
		targets := incident.AffectedComponentTargets
		if len(targets) == 0 {
			targets = make([]models.IncidentAffectedComponent, 0, len(incident.AffectedComponents))
			for _, componentID := range incident.AffectedComponents {
				targets = append(targets, models.IncidentAffectedComponent{ComponentID: componentID})
			}
		}

		expandedTargets := make([]models.IncidentAffectedComponentExpanded, 0, len(targets))
		expandedComponents := make([]models.Component, 0, len(targets))
		seenComponents := map[primitive.ObjectID]struct{}{}
		for _, target := range targets {
			component, ok := incidentComponentMap[target.ComponentID]
			if !ok {
				continue
			}

			if _, exists := seenComponents[component.ID]; !exists {
				expandedComponents = append(expandedComponents, component)
				seenComponents[component.ID] = struct{}{}
			}

			expandedSubComponents := make([]models.SubComponent, 0, len(target.SubComponentIDs))
			for _, subComponentID := range target.SubComponentIDs {
				if subComponent, exists := incidentSubComponentMap[subComponentID]; exists {
					expandedSubComponents = append(expandedSubComponents, subComponent)
				}
			}

			expandedTargets = append(expandedTargets, models.IncidentAffectedComponentExpanded{
				Component:     component,
				SubComponents: expandedSubComponents,
			})
		}

		incidentsWithUpdates = append(incidentsWithUpdates, models.IncidentWithUpdates{
			Incident:                 incident,
			Updates:                  updatesByIncident[incident.ID],
			AffectedComponents:       expandedComponents,
			AffectedComponentTargets: expandedTargets,
		})
	}

	return &CategorySummary{
		Prefix:          componentPrefix(categoryComponent.Name),
		Name:            categoryComponent.Name,
		Description:     categoryComponent.Description,
		AggregateStatus: aggregateStatus,
		Uptime90d:       categoryUptime,
		Services:        services,
		Incidents:       incidentsWithUpdates,
	}, nil
}

func (s *Service) BuildSummary(ctx context.Context, now time.Time) (*StatusSummary, error) {
	components, err := s.repo.ListComponents(ctx)
	if err != nil {
		return nil, err
	}

	subComponents, err := s.repo.ListAllSubComponents(ctx)
	if err != nil {
		return nil, err
	}

	activeIncidents, err := s.repo.ListActiveIncidents(ctx)
	if err != nil {
		return nil, err
	}

	activeMaintenance, err := s.repo.ListActiveMaintenanceAt(ctx, now)
	if err != nil {
		return nil, err
	}

	derivedStatuses := deriveStatuses(components, subComponents, activeIncidents, activeMaintenance)

	counts := map[string]int{
		"operational":          0,
		"degraded_performance": 0,
		"partial_outage":       0,
		"major_outage":         0,
		"maintenance":          0,
	}
	for _, component := range components {
		counts[string(derivedStatuses.ComponentStatus[component.ID])]++
	}

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

	activeCount, err := s.repo.CountActiveIncidents(ctx)
	if err != nil {
		return nil, err
	}

	maintenanceCount, err := s.repo.CountActiveMaintenanceAt(ctx, now)
	if err != nil {
		return nil, err
	}

	return &StatusSummary{
		OverallStatus:        overall,
		ComponentCounts:      counts,
		ActiveIncidents:      int(activeCount),
		ScheduledMaintenance: int(maintenanceCount),
	}, nil
}

func (s *Service) BuildComponents(ctx context.Context, now time.Time) ([]ComponentWithSubStatusInfo, error) {
	components, err := s.repo.ListComponents(ctx)
	if err != nil {
		return nil, err
	}

	allSubs, err := s.repo.ListAllSubComponents(ctx)
	if err != nil {
		return nil, err
	}

	componentIDs := make([]primitive.ObjectID, 0, len(components))
	for _, component := range components {
		componentIDs = append(componentIDs, component.ID)
	}

	subComponentIDs := make([]primitive.ObjectID, 0, len(allSubs))
	subComponentToComponent := make(map[primitive.ObjectID]primitive.ObjectID, len(allSubs))
	for _, sub := range allSubs {
		subComponentIDs = append(subComponentIDs, sub.ID)
		subComponentToComponent[sub.ID] = sub.ComponentID
	}

	monitors, err := s.repo.ListMonitorsByTargets(ctx, componentIDs, subComponentIDs)
	if err != nil {
		return nil, err
	}

	monitorIDs := make([]primitive.ObjectID, 0, len(monitors))
	for _, monitor := range monitors {
		monitorIDs = append(monitorIDs, monitor.ID)
	}

	uptimeRecords, err := s.repo.ListDailyUptimeSinceByMonitorIDs(ctx, monitorIDs, now.AddDate(0, 0, -90))
	if err != nil {
		return nil, err
	}

	uptimeByMonitor := make(map[primitive.ObjectID][]models.DailyUptime)
	for _, record := range uptimeRecords {
		uptimeByMonitor[record.MonitorID] = append(uptimeByMonitor[record.MonitorID], record)
	}

	monitorsByComp := make(map[primitive.ObjectID][]primitive.ObjectID, len(components))
	for _, monitor := range monitors {
		if !monitor.ComponentID.IsZero() {
			monitorsByComp[monitor.ComponentID] = append(monitorsByComp[monitor.ComponentID], monitor.ID)
		}
		if !monitor.SubComponentID.IsZero() {
			if componentID, ok := subComponentToComponent[monitor.SubComponentID]; ok {
				monitorsByComp[componentID] = append(monitorsByComp[componentID], monitor.ID)
			}
		}
	}

	activeIncidents, err := s.repo.ListActiveIncidents(ctx)
	if err != nil {
		return nil, err
	}

	activeMaintenance, err := s.repo.ListActiveMaintenanceAt(ctx, now)
	if err != nil {
		return nil, err
	}

	derivedStatuses := deriveStatuses(components, allSubs, activeIncidents, activeMaintenance)

	displayStatuses := deriveComponentDisplayStatuses(components, allSubs, activeIncidents, activeMaintenance, derivedStatuses)
	for i := range components {
		if status, ok := displayStatuses[components[i].ID]; ok {
			components[i].Status = status
		}
	}

	for i := range allSubs {
		if status, ok := derivedStatuses.SubStatus[allSubs[i].ID]; ok {
			allSubs[i].Status = status
		}
	}

	subsByComp := make(map[primitive.ObjectID][]models.SubComponent)
	for _, sub := range allSubs {
		subsByComp[sub.ComponentID] = append(subsByComp[sub.ComponentID], sub)
	}

	result := make([]ComponentWithSubStatusInfo, 0, len(components))
	for _, component := range components {
		subs := subsByComp[component.ID]
		if subs == nil {
			subs = []models.SubComponent{}
		}

		bars := build90DayBars(monitorsByComp[component.ID], uptimeByMonitor)
		lastIncident, err := s.repo.FindLatestIncidentByComponent(ctx, component.ID)
		if err != nil {
			return nil, err
		}

		result = append(result, ComponentWithSubStatusInfo{
			Component:     component,
			SubComponents: subs,
			UptimeHistory: bars,
			LastIncident:  buildIncidentStatusInfo(lastIncident, now),
		})
	}

	if result == nil {
		result = []ComponentWithSubStatusInfo{}
	}

	return result, nil
}

func (s *Service) BuildIncidents(ctx context.Context, now time.Time, startDate, endDate *time.Time) (*IncidentsResponse, error) {
	var incidents []models.Incident
	var err error

	if startDate != nil || endDate != nil {
		start := time.Time{}
		if startDate != nil {
			start = *startDate
		}

		end := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
		if endDate != nil {
			end = *endDate
		}

		incidents, err = s.repo.ListIncidentsByCreatedAtRange(ctx, start, end)
		if err != nil {
			return nil, err
		}
	} else {
		activeIncidents, err := s.repo.ListActiveIncidents(ctx)
		if err != nil {
			return nil, err
		}

		resolvedIncidents, err := s.repo.ListResolvedIncidentsSince(ctx, now.AddDate(0, 0, -30))
		if err != nil {
			return nil, err
		}

		incidents = append(activeIncidents, resolvedIncidents...)
	}

	if incidents == nil {
		incidents = []models.Incident{}
	}

	incidentIDs := make([]primitive.ObjectID, 0, len(incidents))
	componentIDSet := make(map[primitive.ObjectID]struct{})
	subComponentIDSet := make(map[primitive.ObjectID]struct{})

	for _, incident := range incidents {
		incidentIDs = append(incidentIDs, incident.ID)
		for _, componentID := range incident.AffectedComponents {
			componentIDSet[componentID] = struct{}{}
		}
		for _, target := range normalizeIncidentTargetsForExpansion(incident) {
			componentIDSet[target.ComponentID] = struct{}{}
			for _, subComponentID := range target.SubComponentIDs {
				subComponentIDSet[subComponentID] = struct{}{}
			}
		}
	}

	updatesMap, err := s.repo.ListIncidentUpdatesByIncidentIDs(ctx, incidentIDs)
	if err != nil {
		return nil, err
	}

	componentIDs := make([]primitive.ObjectID, 0, len(componentIDSet))
	for id := range componentIDSet {
		componentIDs = append(componentIDs, id)
	}

	subComponentIDs := make([]primitive.ObjectID, 0, len(subComponentIDSet))
	for id := range subComponentIDSet {
		subComponentIDs = append(subComponentIDs, id)
	}

	components, err := s.repo.ListComponentsByIDs(ctx, componentIDs)
	if err != nil {
		return nil, err
	}

	subComponents, err := s.repo.ListSubComponentsByIDs(ctx, subComponentIDs)
	if err != nil {
		return nil, err
	}

	componentMap := make(map[primitive.ObjectID]models.Component, len(components))
	for _, component := range components {
		componentMap[component.ID] = component
	}

	subComponentMap := make(map[primitive.ObjectID]models.SubComponent, len(subComponents))
	for _, subComponent := range subComponents {
		subComponentMap[subComponent.ID] = subComponent
	}

	activeWithUpdates := make([]models.IncidentWithUpdates, 0)
	resolvedWithUpdates := make([]models.IncidentWithUpdates, 0)

	for _, incident := range incidents {
		targets := normalizeIncidentTargetsForExpansion(incident)

		affectedComponentIDs := make([]primitive.ObjectID, 0, len(incident.AffectedComponents)+len(targets))
		affectedComponentIDs = append(affectedComponentIDs, incident.AffectedComponents...)
		for _, target := range targets {
			affectedComponentIDs = append(affectedComponentIDs, target.ComponentID)
		}
		affectedComponentIDs = idsdomain.DedupeObjectIDs(affectedComponentIDs)

		expandedComponents := make([]models.Component, 0, len(affectedComponentIDs))
		for _, componentID := range affectedComponentIDs {
			if component, ok := componentMap[componentID]; ok {
				expandedComponents = append(expandedComponents, component)
			}
		}

		expandedTargets := make([]models.IncidentAffectedComponentExpanded, 0, len(targets))
		for _, target := range targets {
			component, ok := componentMap[target.ComponentID]
			if !ok {
				continue
			}

			expandedSubComponents := make([]models.SubComponent, 0, len(target.SubComponentIDs))
			for _, subComponentID := range target.SubComponentIDs {
				if subComponent, ok := subComponentMap[subComponentID]; ok {
					expandedSubComponents = append(expandedSubComponents, subComponent)
				}
			}

			expandedTargets = append(expandedTargets, models.IncidentAffectedComponentExpanded{
				Component:     component,
				SubComponents: expandedSubComponents,
			})
		}

		item := models.IncidentWithUpdates{
			Incident:                 incident,
			Updates:                  updatesMap[incident.ID],
			AffectedComponents:       expandedComponents,
			AffectedComponentTargets: expandedTargets,
		}

		if incident.Status == models.IncidentResolved {
			resolvedWithUpdates = append(resolvedWithUpdates, item)
		} else {
			activeWithUpdates = append(activeWithUpdates, item)
		}
	}

	return &IncidentsResponse{
		Active:   activeWithUpdates,
		Resolved: resolvedWithUpdates,
	}, nil
}

var ErrCategoryNotFound = fmt.Errorf("category not found")

func deriveStatuses(
	components []models.Component,
	subs []models.SubComponent,
	activeIncidents []models.Incident,
	activeMaintenance []models.Maintenance,
) derivedStatusResult {
	componentStatus := make(map[primitive.ObjectID]models.ComponentStatus, len(components))
	subStatus := make(map[primitive.ObjectID]models.ComponentStatus, len(subs))

	for _, component := range components {
		componentStatus[component.ID] = component.Status
	}
	for _, sub := range subs {
		subStatus[sub.ID] = sub.Status
	}

	subsByComp := make(map[primitive.ObjectID][]primitive.ObjectID)
	for _, sub := range subs {
		subsByComp[sub.ComponentID] = append(subsByComp[sub.ComponentID], sub.ID)
	}

	aggByComponent := make(map[primitive.ObjectID]*componentAggregationState, len(components))
	for _, component := range components {
		aggByComponent[component.ID] = &componentAggregationState{
			DirectStatus:   models.StatusOperational,
			ImpactedSubIDs: map[primitive.ObjectID]struct{}{},
		}
	}

	for _, incident := range activeIncidents {
		incidentStatus := mapIncidentImpactToStatus(incident.Impact)
		if incidentStatus == models.StatusOperational {
			continue
		}

		for _, target := range normalizeIncidentTargetsForExpansion(incident) {
			state, ok := aggByComponent[target.ComponentID]
			if !ok {
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
			state, ok := aggByComponent[componentID]
			if !ok {
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

	for _, component := range components {
		state := aggByComponent[component.ID]
		if state == nil {
			continue
		}

		totalSubs := len(subsByComp[component.ID])
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

		componentStatus[component.ID] = derived
	}

	return derivedStatusResult{ComponentStatus: componentStatus, SubStatus: subStatus}
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

func deriveComponentDisplayStatuses(
	components []models.Component,
	subs []models.SubComponent,
	activeIncidents []models.Incident,
	activeMaintenance []models.Maintenance,
	base derivedStatusResult,
) map[primitive.ObjectID]models.ComponentStatus {
	statuses := make(map[primitive.ObjectID]models.ComponentStatus, len(base.ComponentStatus))
	maps.Copy(statuses, base.ComponentStatus)

	subsByComp := make(map[primitive.ObjectID][]primitive.ObjectID)
	for _, sub := range subs {
		subsByComp[sub.ComponentID] = append(subsByComp[sub.ComponentID], sub.ID)
	}

	aggByComponent := make(map[primitive.ObjectID]*componentAggregationState, len(components))
	for _, component := range components {
		aggByComponent[component.ID] = &componentAggregationState{
			DirectStatus:   models.StatusOperational,
			ImpactedSubIDs: map[primitive.ObjectID]struct{}{},
		}
	}

	for _, incident := range activeIncidents {
		incidentStatus := mapIncidentImpactToStatus(incident.Impact)
		if incidentStatus == models.StatusOperational {
			continue
		}

		for _, target := range normalizeIncidentTargetsForExpansion(incident) {
			state, ok := aggByComponent[target.ComponentID]
			if !ok {
				continue
			}

			if len(target.SubComponentIDs) == 0 {
				state.HasDirectImpact = true
				state.DirectStatus = maxStatus(state.DirectStatus, incidentStatus)
				for _, subID := range subsByComp[target.ComponentID] {
					state.ImpactedSubIDs[subID] = struct{}{}
				}
				continue
			}

			for _, subID := range target.SubComponentIDs {
				state.ImpactedSubIDs[subID] = struct{}{}
			}
		}
	}

	for _, maintenance := range activeMaintenance {
		for _, componentID := range maintenance.Components {
			state, ok := aggByComponent[componentID]
			if !ok {
				continue
			}

			state.HasDirectImpact = true
			state.DirectStatus = maxStatus(state.DirectStatus, models.StatusMaintenance)
		}
	}

	for componentID, state := range aggByComponent {
		if state == nil || state.HasDirectImpact || len(state.ImpactedSubIDs) == 0 {
			continue
		}

		worstSubStatus := models.StatusOperational
		for subID := range state.ImpactedSubIDs {
			worstSubStatus = maxStatus(worstSubStatus, base.SubStatus[subID])
		}
		statuses[componentID] = partialAggregateStatusFromSubStatus(worstSubStatus)
	}

	return statuses
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

func buildIncidentStatusInfo(incident *models.Incident, now time.Time) *IncidentStatusInfo {
	if incident == nil {
		return nil
	}

	durationStr := "Unknown"
	if incident.ResolvedAt != nil && !incident.ResolvedAt.IsZero() {
		durationStr = formatIncidentDuration(incident.ResolvedAt.Sub(incident.CreatedAt))
	} else {
		durationStr = formatIncidentDuration(now.Sub(incident.CreatedAt))
	}

	return &IncidentStatusInfo{
		Date:     incident.CreatedAt,
		Duration: durationStr,
		Title:    incident.Title,
	}
}

func formatIncidentDuration(duration time.Duration) string {
	if duration.Minutes() < 60 {
		return fmt.Sprintf("%.0f minute(s)", duration.Minutes())
	}
	if duration.Hours() < 24 {
		return fmt.Sprintf("%.1f hour(s)", duration.Hours())
	}
	return fmt.Sprintf("%.1f day(s)", duration.Hours()/24)
}

func findCategoryComponent(components []models.Component, prefix string) *models.Component {
	normalizedPrefix := normalizeCategoryPrefix(prefix)
	if normalizedPrefix == "" {
		return nil
	}

	for i := range components {
		if componentPrefix(components[i].Name) == normalizedPrefix {
			return &components[i]
		}
	}

	for i := range components {
		if strings.HasPrefix(componentPrefix(components[i].Name), normalizedPrefix) {
			return &components[i]
		}
	}

	return nil
}

func normalizeCategoryPrefix(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}

	parts := strings.FieldsFunc(v, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "-")
}

func componentPrefix(name string) string {
	return normalizeCategoryPrefix(name)
}

func build90DayBars(monitorIDs []primitive.ObjectID, uptimeByMonitorID map[primitive.ObjectID][]models.DailyUptime) []DailyUptimeBar {
	domainBars := uptimedomain.Build90DayBars(monitorIDs, uptimeByMonitorID)
	bars := make([]DailyUptimeBar, 0, len(domainBars))
	for _, bar := range domainBars {
		bars = append(bars, DailyUptimeBar{
			Date:          bar.Date,
			UptimePercent: bar.UptimePercent,
			Status:        string(bar.Status),
		})
	}
	return bars
}

func averageUptime(bars []DailyUptimeBar) float64 {
	if len(bars) == 0 {
		return 0
	}

	total := 0.0
	for _, bar := range bars {
		total += bar.UptimePercent
	}

	return total / float64(len(bars))
}

func aggregateStatusFromServices(services []CategoryService) string {
	if len(services) == 0 {
		return string(models.StatusOperational)
	}

	statuses := make([]models.ComponentStatus, 0, len(services))
	for _, service := range services {
		statuses = append(statuses, service.Status)
	}

	sort.SliceStable(statuses, func(i, j int) bool {
		return statusdomain.ComponentSeverityRank(statuses[i]) > statusdomain.ComponentSeverityRank(statuses[j])
	})

	return string(statuses[0])
}
