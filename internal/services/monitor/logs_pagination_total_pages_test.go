package monitor

import "testing"

func TestGetMonitorLogsPaginatedTotalPagesRoundsUp(t *testing.T) {
	if got := calculateTotalPages(101, 10); got != 11 {
		t.Fatalf("expected 11 total pages, got %d", got)
	}
}

func TestGetMonitorLogsPaginatedTotalPagesReturnsZeroForEmptyTotal(t *testing.T) {
	if got := calculateTotalPages(0, 10); got != 0 {
		t.Fatalf("expected 0 total pages, got %d", got)
	}
}
