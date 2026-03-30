package subscriber

import (
	"context"
	"fmt"

	shared "github.com/fresp/StatusForge/internal/domain/shared"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	repo repository.SubscriberRepository
}

func NewService(repo repository.SubscriberRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, email string) (models.Subscriber, error) {
	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return models.Subscriber{}, err
	}
	if existing != nil {
		return models.Subscriber{}, fmt.Errorf("%w: email already subscribed", shared.ErrConflict)
	}

	sub := repository.NewSubscriber(email)
	if err := s.repo.Insert(ctx, sub); err != nil {
		return models.Subscriber{}, err
	}

	return sub, nil
}

func (s *Service) List(ctx context.Context, page, limit int) ([]models.Subscriber, int64, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *Service) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	deleted, err := s.repo.DeleteByID(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("%w: subscriber not found", shared.ErrNotFound)
	}

	return nil
}
