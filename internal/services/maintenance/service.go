package maintenance

import (
	"context"
	"errors"
	"fmt"
	"time"

	shared "github.com/fresp/StatusForge/internal/domain/shared"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Service struct {
	repo repository.MaintenanceRepository
}

func NewService(repo repository.MaintenanceRepository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	Title           string
	Description     string
	Components      []string
	StartTime       string
	EndTime         string
	CreatorIDHex    string
	CreatorUsername string
}

type UpdateInput struct {
	Title       string
	Description string
	Status      models.MaintenanceStatus
	StartTime   string
	EndTime     string
}

func (s *Service) List(ctx context.Context, page, limit int) ([]models.Maintenance, int64, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *Service) ListPublic(ctx context.Context, page, limit int) ([]models.Maintenance, int64, error) {
	return s.repo.ListPublic(ctx, page, limit)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (models.Maintenance, error) {
	startTime, err := time.Parse(time.RFC3339, input.StartTime)
	if err != nil {
		return models.Maintenance{}, fmt.Errorf("%w: invalid startTime format, use RFC3339", shared.ErrInvalidInput)
	}
	endTime, err := time.Parse(time.RFC3339, input.EndTime)
	if err != nil {
		return models.Maintenance{}, fmt.Errorf("%w: invalid endTime format, use RFC3339", shared.ErrInvalidInput)
	}

	creatorID, err := primitive.ObjectIDFromHex(input.CreatorIDHex)
	if err != nil {
		return models.Maintenance{}, fmt.Errorf("%w: invalid authenticated user id", shared.ErrUnauthorized)
	}

	componentIDs := make([]primitive.ObjectID, 0, len(input.Components))
	for _, raw := range input.Components {
		oid, parseErr := primitive.ObjectIDFromHex(raw)
		if parseErr == nil {
			componentIDs = append(componentIDs, oid)
		}
	}

	status := models.MaintenanceScheduled
	if time.Now().After(startTime) {
		status = models.MaintenanceInProgress
	}

	maintenance := models.Maintenance{
		ID:              primitive.NewObjectID(),
		Title:           input.Title,
		Description:     input.Description,
		CreatorID:       &creatorID,
		CreatorUsername: input.CreatorUsername,
		Components:      componentIDs,
		StartTime:       startTime,
		EndTime:         endTime,
		Status:          status,
	}

	if err := s.repo.Insert(ctx, maintenance); err != nil {
		return models.Maintenance{}, err
	}

	return maintenance, nil
}

func (s *Service) Update(ctx context.Context, id primitive.ObjectID, input UpdateInput) (models.Maintenance, error) {
	setFields := bson.M{}
	if input.Title != "" {
		setFields["title"] = input.Title
	}
	if input.Description != "" {
		setFields["description"] = input.Description
	}
	if input.Status != "" {
		setFields["status"] = input.Status
	}
	if input.StartTime != "" {
		t, err := time.Parse(time.RFC3339, input.StartTime)
		if err != nil {
			return models.Maintenance{}, fmt.Errorf("%w: invalid startTime format, use RFC3339", shared.ErrInvalidInput)
		}
		setFields["startTime"] = t
	}
	if input.EndTime != "" {
		t, err := time.Parse(time.RFC3339, input.EndTime)
		if err != nil {
			return models.Maintenance{}, fmt.Errorf("%w: invalid endTime format, use RFC3339", shared.ErrInvalidInput)
		}
		setFields["endTime"] = t
	}

	updated, err := s.repo.UpdateByID(ctx, id, setFields)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.Maintenance{}, fmt.Errorf("%w: maintenance not found", shared.ErrNotFound)
		}
		return models.Maintenance{}, err
	}

	return updated, nil
}
