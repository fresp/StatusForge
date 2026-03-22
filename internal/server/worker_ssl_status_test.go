package server

import "testing"

func TestShouldRestoreOperational(t *testing.T) {
	tests := []struct {
		name              string
		hasActiveOutage   bool
		hasActiveIncident bool
		want              bool
	}{
		{
			name:              "restore when no active outage or incident",
			hasActiveOutage:   false,
			hasActiveIncident: false,
			want:              true,
		},
		{
			name:              "do not restore when outage is active",
			hasActiveOutage:   true,
			hasActiveIncident: false,
			want:              false,
		},
		{
			name:              "do not restore when incident is active",
			hasActiveOutage:   false,
			hasActiveIncident: true,
			want:              false,
		},
		{
			name:              "do not restore when both are active",
			hasActiveOutage:   true,
			hasActiveIncident: true,
			want:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRestoreOperational(tt.hasActiveOutage, tt.hasActiveIncident)
			if got != tt.want {
				t.Fatalf("shouldRestoreOperational() = %v, want %v", got, tt.want)
			}
		})
	}
}
