package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDateRangeParamsAcceptsDateOnlyBoundaries(t *testing.T) {
	start, end, err := parseDateRangeParams("2026-01-01", "2026-03-31")

	require.NoError(t, err)
	require.NotNil(t, start)
	require.NotNil(t, end)

	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), *start)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), *end)
}

func TestParseDateRangeParamsAcceptsRFC3339(t *testing.T) {
	start, end, err := parseDateRangeParams("2026-01-01T07:00:00+07:00", "2026-03-31T23:00:00+07:00")

	require.NoError(t, err)
	require.NotNil(t, start)
	require.NotNil(t, end)

	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), *start)
	assert.Equal(t, time.Date(2026, 3, 31, 16, 0, 0, 0, time.UTC), *end)
}

func TestParseDateRangeParamsRejectsInvalidDate(t *testing.T) {
	start, end, err := parseDateRangeParams("bad-date", "2026-03-31")

	assert.Nil(t, start)
	assert.Nil(t, end)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start_date")
}

func TestParseDateRangeParamsRejectsStartAfterEnd(t *testing.T) {
	start, end, err := parseDateRangeParams("2026-04-01", "2026-03-31")

	assert.Nil(t, start)
	assert.Nil(t, end)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start_date must be before or equal to end_date")
}
