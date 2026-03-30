package webhook

import (
	"context"
	"fmt"
	"time"

	shared "github.com/fresp/StatusForge/internal/domain/shared"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	repo repository.WebhookChannelRepository
}

func NewService(repo repository.WebhookChannelRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, page, limit int) ([]models.WebhookChannel, int64, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *Service) Create(ctx context.Context, name, url string) (models.WebhookChannel, error) {
	channel := models.WebhookChannel{
		ID:        primitive.NewObjectID(),
		Name:      name,
		URL:       url,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Insert(ctx, channel); err != nil {
		return models.WebhookChannel{}, err
	}

	return channel, nil
}

func (s *Service) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	deleted, err := s.repo.DeleteByID(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("%w: webhook channel not found", shared.ErrNotFound)
	}

	return nil
}
