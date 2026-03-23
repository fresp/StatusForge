package handlers

import (
	"fmt"
	"time"
)

const (
	dateOnlyLayout = "2006-01-02"
)

func parseDateRangeParams(startDateRaw, endDateRaw string) (*time.Time, *time.Time, error) {
	var startPtr *time.Time
	var endPtr *time.Time

	if startDateRaw != "" {
		parsedStart, err := parseBoundaryDate(startDateRaw, true)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid start_date: %w", err)
		}
		startPtr = &parsedStart
	}

	if endDateRaw != "" {
		parsedEnd, err := parseBoundaryDate(endDateRaw, false)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid end_date: %w", err)
		}
		endPtr = &parsedEnd
	}

	if startPtr != nil && endPtr != nil && !startPtr.Before(*endPtr) {
		return nil, nil, fmt.Errorf("start_date must be before or equal to end_date")
	}

	return startPtr, endPtr, nil
}

func parseBoundaryDate(raw string, isStart bool) (time.Time, error) {
	if parsedDateOnly, err := time.Parse(dateOnlyLayout, raw); err == nil {
		if !isStart {
			return parsedDateOnly.UTC().Add(24 * time.Hour), nil
		}
		return parsedDateOnly.UTC(), nil
	}

	parsedDateTime, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DD")
	}

	return parsedDateTime.UTC(), nil
}
