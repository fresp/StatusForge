package incident

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	idsdomain "github.com/fresp/StatusForge/internal/domain/ids"
	shared "github.com/fresp/StatusForge/internal/domain/shared"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var errInvalidAffectedComponentPayload = errors.New("invalid affected component payload")

var (
	ErrInvalidAffectedComponentsPayload   = errors.New("invalid affected components payload")
	ErrInvalidAffectedComponentsReference = errors.New("one or more affected components or subcomponents are invalid")
)

type AffectedComponentInput struct {
	ComponentID     string
	SubComponentIDs []string
}

type RequestBody struct {
	Title                    string
	Description              string
	Status                   models.IncidentStatus
	Impact                   models.IncidentImpact
	AffectedComponents       []string
	Components               []string
	AffectedComponentTargets []AffectedComponentInput
	AffectedComponentsNew    []AffectedComponentInput
}

type CreateInput struct {
	RequestBody
	CreatorIDHex    string
	CreatorUsername string
}

type Service struct {
	repo repository.IncidentRepository
}

func NewService(repo repository.IncidentRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, statusFilter, startDateRaw, endDateRaw string, page, limit int) ([]models.Incident, int64, error) {
	filter := bson.M{}
	if statusFilter == "active" {
		filter["status"] = bson.M{"$ne": models.IncidentResolved}
	} else if statusFilter != "" {
		filter["status"] = statusFilter
	}

	startDate, endDate, err := parseDateRangeParams(startDateRaw, endDateRaw)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", shared.ErrInvalidInput, err)
	}

	if startDate != nil || endDate != nil {
		createdAtFilter := bson.M{}
		if startDate != nil {
			createdAtFilter["$gte"] = *startDate
		}
		if endDate != nil {
			createdAtFilter["$lt"] = *endDate
		}
		filter["createdAt"] = createdAtFilter
	}

	return s.repo.List(ctx, filter, page, limit)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (models.Incident, error) {
	status := input.Status
	if status == "" {
		status = models.IncidentInvestigating
	}

	impact := input.Impact
	if impact == "" {
		impact = models.ImpactMinor
	}

	userID, err := primitive.ObjectIDFromHex(input.CreatorIDHex)
	if err != nil {
		return models.Incident{}, fmt.Errorf("%w: invalid authenticated user id", shared.ErrUnauthorized)
	}

	compIDs, targets, err := normalizeIncidentTargets(
		append(input.AffectedComponentTargets, input.AffectedComponentsNew...),
		input.AffectedComponents,
		input.Components,
		true,
	)
	if err != nil {
		if errors.Is(err, errInvalidAffectedComponentPayload) {
			return models.Incident{}, fmt.Errorf("%w: %w", shared.ErrInvalidInput, ErrInvalidAffectedComponentsPayload)
		}
		return models.Incident{}, fmt.Errorf("%w: %w", shared.ErrInvalidInput, ErrInvalidAffectedComponentsPayload)
	}

	if err := s.validateIncidentTargets(ctx, targets); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.Incident{}, fmt.Errorf("%w: %w", shared.ErrInvalidInput, ErrInvalidAffectedComponentsReference)
		}
		return models.Incident{}, err
	}

	now := time.Now()
	incident := models.Incident{
		ID:                       primitive.NewObjectID(),
		Title:                    input.Title,
		Description:              input.Description,
		Status:                   status,
		Impact:                   impact,
		CreatorID:                &userID,
		CreatorUsername:          input.CreatorUsername,
		AffectedComponents:       compIDs,
		AffectedComponentTargets: targets,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	if err := s.repo.InsertIncident(ctx, incident); err != nil {
		return models.Incident{}, err
	}

	return incident, nil
}

func (s *Service) Update(ctx context.Context, id primitive.ObjectID, input RequestBody) (models.Incident, error) {
	setFields := bson.M{"updatedAt": time.Now()}
	if input.Title != "" {
		setFields["title"] = input.Title
	}
	if input.Description != "" {
		setFields["description"] = input.Description
	}
	if input.Status != "" {
		setFields["status"] = input.Status
		if input.Status == models.IncidentResolved {
			now := time.Now()
			setFields["resolvedAt"] = now
		}
	}
	if input.Impact != "" {
		setFields["impact"] = input.Impact
	}

	hasTargetPayload := len(input.AffectedComponentTargets) > 0 || len(input.AffectedComponentsNew) > 0 || len(input.AffectedComponents) > 0 || len(input.Components) > 0
	if hasTargetPayload {
		compIDs, targets, err := normalizeIncidentTargets(
			append(input.AffectedComponentTargets, input.AffectedComponentsNew...),
			input.AffectedComponents,
			input.Components,
			true,
		)
		if err != nil {
			return models.Incident{}, fmt.Errorf("%w: %w", shared.ErrInvalidInput, ErrInvalidAffectedComponentsPayload)
		}

		if err := s.validateIncidentTargets(ctx, targets); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return models.Incident{}, fmt.Errorf("%w: %w", shared.ErrInvalidInput, ErrInvalidAffectedComponentsReference)
			}
			return models.Incident{}, err
		}

		setFields["affectedComponents"] = compIDs
		setFields["affectedComponentTargets"] = targets
	}

	incident, err := s.repo.UpdateIncidentByID(ctx, id, setFields)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.Incident{}, fmt.Errorf("%w: incident not found", shared.ErrNotFound)
		}
		return models.Incident{}, err
	}

	return incident, nil
}

func (s *Service) AddUpdate(ctx context.Context, incidentID primitive.ObjectID, message string, status models.IncidentStatus) (models.IncidentUpdate, error) {
	update := models.IncidentUpdate{
		ID:         primitive.NewObjectID(),
		IncidentID: incidentID,
		Message:    message,
		Status:     status,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.InsertUpdate(ctx, update); err != nil {
		return models.IncidentUpdate{}, err
	}

	if err := s.repo.ApplyIncidentStatus(ctx, incidentID, status); err != nil {
		return models.IncidentUpdate{}, err
	}

	return update, nil
}

func (s *Service) ListUpdates(ctx context.Context, incidentID primitive.ObjectID) ([]models.IncidentUpdate, error) {
	return s.repo.ListUpdates(ctx, incidentID)
}

func (s *Service) validateIncidentTargets(ctx context.Context, targets []models.IncidentAffectedComponent) error {
	if len(targets) == 0 {
		return nil
	}

	componentIDs := make([]primitive.ObjectID, 0, len(targets))
	for _, t := range targets {
		componentIDs = append(componentIDs, t.ComponentID)
	}

	componentCount, err := s.repo.CountComponents(ctx, componentIDs)
	if err != nil {
		return err
	}
	if componentCount != int64(len(idsdomain.DedupeObjectIDs(componentIDs))) {
		return mongo.ErrNoDocuments
	}

	for _, t := range targets {
		if len(t.SubComponentIDs) == 0 {
			continue
		}
		subCount, subErr := s.repo.CountSubComponentsByComponent(ctx, t.ComponentID, t.SubComponentIDs)
		if subErr != nil {
			return subErr
		}
		if subCount != int64(len(idsdomain.DedupeObjectIDs(t.SubComponentIDs))) {
			return mongo.ErrNoDocuments
		}
	}

	return nil
}

func normalizeIncidentTargets(
	targets []AffectedComponentInput,
	legacy []string,
	legacyAlias []string,
	allowEmpty bool,
) ([]primitive.ObjectID, []models.IncidentAffectedComponent, error) {
	bucket := make(map[primitive.ObjectID]map[primitive.ObjectID]struct{})
	componentOrder := make([]primitive.ObjectID, 0)

	mergeTarget := func(componentID primitive.ObjectID, subIDs []primitive.ObjectID) {
		if _, ok := bucket[componentID]; !ok {
			bucket[componentID] = map[primitive.ObjectID]struct{}{}
			componentOrder = append(componentOrder, componentID)
		}
		for _, subID := range subIDs {
			bucket[componentID][subID] = struct{}{}
		}
	}

	for _, t := range targets {
		if t.ComponentID == "" {
			return nil, nil, errInvalidAffectedComponentPayload
		}
		componentID, err := primitive.ObjectIDFromHex(t.ComponentID)
		if err != nil {
			return nil, nil, err
		}

		subIDs := make([]primitive.ObjectID, 0, len(t.SubComponentIDs))
		for _, sid := range t.SubComponentIDs {
			subID, subErr := primitive.ObjectIDFromHex(sid)
			if subErr != nil {
				return nil, nil, subErr
			}
			subIDs = append(subIDs, subID)
		}
		mergeTarget(componentID, subIDs)
	}

	for _, rawID := range append(legacy, legacyAlias...) {
		componentID, err := primitive.ObjectIDFromHex(rawID)
		if err != nil {
			return nil, nil, err
		}
		mergeTarget(componentID, nil)
	}

	componentIDs := make([]primitive.ObjectID, 0, len(componentOrder))
	normalizedTargets := make([]models.IncidentAffectedComponent, 0, len(componentOrder))

	for _, componentID := range componentOrder {
		subMap := bucket[componentID]
		subIDs := make([]primitive.ObjectID, 0, len(subMap))
		for sid := range subMap {
			subIDs = append(subIDs, sid)
		}
		sort.Slice(subIDs, func(i, j int) bool { return subIDs[i].Hex() < subIDs[j].Hex() })

		componentIDs = append(componentIDs, componentID)
		normalizedTargets = append(normalizedTargets, models.IncidentAffectedComponent{
			ComponentID:     componentID,
			SubComponentIDs: subIDs,
		})
	}

	componentIDs = idsdomain.DedupeObjectIDs(componentIDs)

	if !allowEmpty && len(componentIDs) == 0 {
		return nil, nil, mongo.ErrNoDocuments
	}

	return componentIDs, normalizedTargets, nil
}

const dateOnlyLayout = "2006-01-02"

func parseDateRangeParams(startDateRaw, endDateRaw string) (*time.Time, *time.Time, error) {
	var startPtr *time.Time
	var endPtr *time.Time

	if startDateRaw != "" {
		parsedStart, err := parseBoundaryDate(startDateRaw, true)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid start_date: %w", err)
		}
		startPtr = &parsedStart
	}

	if endDateRaw != "" {
		parsedEnd, err := parseBoundaryDate(endDateRaw, false)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid end_date: %w", err)
		}
		endPtr = &parsedEnd
	}

	if startPtr != nil && endPtr != nil && !startPtr.Before(*endPtr) {
		return nil, nil, fmt.Errorf("start_date must be before or equal to end_date")
	}

	return startPtr, endPtr, nil
}

func parseBoundaryDate(raw string, isStart bool) (time.Time, error) {
	if parsedDateOnly, err := time.Parse(dateOnlyLayout, raw); err == nil {
		if !isStart {
			return parsedDateOnly.UTC().Add(24 * time.Hour), nil
		}
		return parsedDateOnly.UTC(), nil
	}

	parsedDateTime, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DD")
	}

	return parsedDateTime.UTC(), nil
}
