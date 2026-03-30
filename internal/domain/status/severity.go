package status

import "github.com/fresp/StatusForge/internal/models"

func ComponentSeverityRank(componentStatus models.ComponentStatus) int {
	switch componentStatus {
	case models.StatusMajorOutage:
		return 5
	case models.StatusPartialOutage:
		return 4
	case models.StatusDegradedPerf:
		return 3
	case models.StatusMaintenance:
		return 2
	case models.StatusOperational:
		return 1
	default:
		return 0
	}
}

func MaxComponentStatus(a, b models.ComponentStatus) models.ComponentStatus {
	if ComponentSeverityRank(a) >= ComponentSeverityRank(b) {
		return a
	}
	return b
}
