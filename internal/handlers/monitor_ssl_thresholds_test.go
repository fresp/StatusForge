package handlers

import (
	"reflect"
	"testing"

	monitorservice "github.com/fresp/Statora/internal/services/monitor"
)

func TestSanitizeSSLThresholds_DefaultWhenEmpty(t *testing.T) {
	got := monitorservice.SanitizeSSLThresholds(nil)
	want := []int{30, 14, 7}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sanitizeSSLThresholds(nil) = %v, want %v", got, want)
	}
}

func TestSanitizeSSLThresholds_FiltersDeduplicatesAndSortsDescending(t *testing.T) {
	got := monitorservice.SanitizeSSLThresholds([]int{7, 30, 14, 14, -1, 0})
	want := []int{30, 14, 7}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sanitizeSSLThresholds(...) = %v, want %v", got, want)
	}
}
