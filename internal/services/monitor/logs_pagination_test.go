package monitor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type monitorLogsPaginationRepoStub struct {
	logs            []models.MonitorLog
	total           int64
	err             error
	called          bool
	calledMonitorID primitive.ObjectID
	calledPage      int
	calledLimit     int
}

func (s *monitorLogsPaginationRepoStub) Insert(_ context.Context, _ models.Monitor) error {
	return errors.New("not implemented")
}

func (s *monitorLogsPaginationRepoStub) Update(_ context.Context, _ primitive.ObjectID, _ models.Monitor) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *monitorLogsPaginationRepoStub) Delete(_ context.Context, _ primitive.ObjectID) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *monitorLogsPaginationRepoStub) List(_ context.Context, _ int, _ int) ([]models.Monitor, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *monitorLogsPaginationRepoStub) ListLogs(_ context.Context, _ primitive.ObjectID, _ int64) ([]models.MonitorLog, error) {
	return nil, errors.New("not implemented")
}

func (s *monitorLogsPaginationRepoStub) ListUptime(_ context.Context, _ primitive.ObjectID, _ time.Time) ([]models.DailyUptime, error) {
	return nil, errors.New("not implemented")
}

func (s *monitorLogsPaginationRepoStub) ListOutages(_ context.Context) ([]models.Outage, error) {
	return nil, errors.New("not implemented")
}

func (s *monitorLogsPaginationRepoStub) ListHistory(_ context.Context, _ primitive.ObjectID, _ int64) ([]models.EnhancedMonitorLog, error) {
	return nil, errors.New("not implemented")
}

func (s *monitorLogsPaginationRepoStub) FindLogsByMonitorIDPaginated(_ context.Context, monitorID primitive.ObjectID, page, limit int) ([]models.MonitorLog, int64, error) {
	s.called = true
	s.calledMonitorID = monitorID
	s.calledPage = page
	s.calledLimit = limit
	if s.err != nil {
		return nil, 0, s.err
	}
	return s.logs, s.total, nil
}

func TestGetMonitorLogsPaginatedReturnsPaginationMetadata(t *testing.T) {
	monitorID := primitive.NewObjectID()
	repo := &monitorLogsPaginationRepoStub{
		logs:  []models.MonitorLog{{ID: primitive.NewObjectID()}, {ID: primitive.NewObjectID()}},
		total: 101,
	}
	svc := NewService(repo)

	result, err := svc.GetMonitorLogsPaginated(context.Background(), monitorID, 2, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.called {
		t.Fatalf("expected repository ListLogsPaginated to be called")
	}
	if repo.calledMonitorID != monitorID {
		t.Fatalf("expected monitor ID %s, got %s", monitorID.Hex(), repo.calledMonitorID.Hex())
	}
	if repo.calledPage != 2 {
		t.Fatalf("expected page 2, got %d", repo.calledPage)
	}
	if repo.calledLimit != 10 {
		t.Fatalf("expected limit 10, got %d", repo.calledLimit)
	}

	if result.Page != 2 {
		t.Fatalf("expected page 2, got %d", result.Page)
	}
	if result.Total != 101 {
		t.Fatalf("expected total 101, got %d", result.Total)
	}
	if result.TotalPages != 11 {
		t.Fatalf("expected total pages 11, got %d", result.TotalPages)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
}

func TestGetMonitorLogsPaginatedPropagatesRepositoryError(t *testing.T) {
	monitorID := primitive.NewObjectID()
	repo := &monitorLogsPaginationRepoStub{err: errors.New("db down")}
	svc := NewService(repo)

	_, err := svc.GetMonitorLogsPaginated(context.Background(), monitorID, 1, 10)
	if err == nil {
		t.Fatalf("expected error")
	}
}
