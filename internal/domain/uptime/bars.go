package uptime

import (
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DailyBar struct {
	Date          string
	UptimePercent float64
	Status        models.ComponentStatus
}

func Build90DayBars(monitorIDs []primitive.ObjectID, uptimeByMonitorID map[primitive.ObjectID][]models.DailyUptime) []DailyBar {
	if len(monitorIDs) == 0 {
		return []DailyBar{}
	}

	bars := make([]DailyBar, 0, 90)
	now := time.Now()

	for i := 89; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayKey := day.Format("2006-01-02")

		totalChecks := 0
		successfulChecks := 0

		for _, monitorID := range monitorIDs {
			for _, record := range uptimeByMonitorID[monitorID] {
				if record.Date.Format("2006-01-02") == dayKey {
					totalChecks += record.TotalChecks
					successfulChecks += record.SuccessfulChecks
				}
			}
		}

		uptimePercent := 100.0
		status := models.StatusOperational

		if totalChecks > 0 {
			uptimePercent = (float64(successfulChecks) / float64(totalChecks)) * 100
			switch {
			case uptimePercent < 50:
				status = models.StatusMajorOutage
			case uptimePercent < 99.9:
				status = models.StatusDegradedPerf
			default:
				status = models.StatusOperational
			}
		}

		bars = append(bars, DailyBar{
			Date:          dayKey,
			UptimePercent: uptimePercent,
			Status:        status,
		})
	}

	return bars
}
