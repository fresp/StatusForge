package subcomponent

import (
	"context"
	"fmt"
	"time"

	shared "github.com/fresp/Statora/internal/domain/shared"
	"github.com/fresp/Statora/internal/models"
	"github.com/fresp/Statora/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Service struct {
	repo repository.SubComponentRepository
}

func NewService(repo repository.SubComponentRepository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	ComponentID string
	Name        string
	Description string
	Status      models.ComponentStatus
}

type UpdateInput struct {
	ComponentID string
	Name        string
	Description string
	Status      models.ComponentStatus
}

func (s *Service) List(ctx context.Context, componentIDHex string, page, limit int) ([]models.SubComponent, int64, error) {
	filter := bson.M{}
	if componentIDHex != "" {
		oid, err := primitive.ObjectIDFromHex(componentIDHex)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: invalid componentId", shared.ErrInvalidInput)
		}
		filter["componentId"] = oid
	}

	return s.repo.List(ctx, filter, page, limit)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (models.SubComponent, error) {
	compID, err := primitive.ObjectIDFromHex(input.ComponentID)
	if err != nil {
		return models.SubComponent{}, fmt.Errorf("%w: invalid componentId", shared.ErrInvalidInput)
	}

	status := input.Status
	if status == "" {
		status = models.StatusOperational
	}

	now := time.Now()
	sub := models.SubComponent{
		ID:          primitive.NewObjectID(),
		ComponentID: compID,
		Name:        input.Name,
		Description: input.Description,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Insert(ctx, sub); err != nil {
		return models.SubComponent{}, err
	}

	return sub, nil
}

func (s *Service) Update(ctx context.Context, id primitive.ObjectID, input UpdateInput) (models.SubComponent, error) {
	if input.ComponentID != "" {
		componentID, err := primitive.ObjectIDFromHex(input.ComponentID)
		if err != nil {
			return models.SubComponent{}, fmt.Errorf("%w: invalid componentId", shared.ErrInvalidInput)
		}

		exists, err := s.repo.ComponentExists(ctx, componentID)
		if err != nil {
			return models.SubComponent{}, err
		}
		if !exists {
			return models.SubComponent{}, fmt.Errorf("%w: component not found", shared.ErrInvalidInput)
		}
	}

	setFields := bson.M{}
	if input.ComponentID != "" {
		componentID, _ := primitive.ObjectIDFromHex(input.ComponentID)
		setFields["componentId"] = componentID
	}
	if input.Name != "" {
		setFields["name"] = input.Name
	}
	if input.Description != "" {
		setFields["description"] = input.Description
	}
	if input.Status != "" {
		setFields["status"] = input.Status
	}
	setFields["updated_at"] = time.Now()

	sub, err := s.repo.UpdateByID(ctx, id, setFields)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.SubComponent{}, fmt.Errorf("%w: subcomponent not found", shared.ErrNotFound)
		}
		return models.SubComponent{}, err
	}

	return sub, nil
}

func (s *Service) Delete(ctx context.Context, id primitive.ObjectID) error {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("%w: subcomponent not found", shared.ErrNotFound)
		}
		return err
	}

	if err := s.repo.CleanupReferencesForDeletedSubComponent(ctx, id, sub.ComponentID); err != nil {
		return err
	}

	deleted, err := s.repo.DeleteByID(ctx, id)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return fmt.Errorf("%w: subcomponent not found", shared.ErrNotFound)
	}

	return nil
}
