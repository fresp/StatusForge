package handlers

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/gin-gonic/gin"
)

const (
	defaultPage      = 1
	defaultLimit     = 20
	logsDefaultLimit = 10
	logsMaxLimit     = 100
)

// PaginationConfig configures pagination parsing behavior.
type PaginationConfig struct {
	DefaultLimit int
	MaxLimit     int // 0 means no maximum
}

// DefaultPaginationConfig returns the standard pagination config (default=20, no max).
func DefaultPaginationConfig() PaginationConfig {
	return PaginationConfig{
		DefaultLimit: defaultLimit,
		MaxLimit:     0,
	}
}

// MonitorLogsPaginationConfig returns the monitor-logs-specific config (default=10, max=100).
func MonitorLogsPaginationConfig() PaginationConfig {
	return PaginationConfig{
		DefaultLimit: logsDefaultLimit,
		MaxLimit:     logsMaxLimit,
	}
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// parsePaginationParams parses page and limit from query params using default config.
// Deprecated: Use ParsePaginationParams with explicit config instead.
func parsePaginationParams(c *gin.Context) (int, int, error) {
	return ParsePaginationParams(c, DefaultPaginationConfig())
}

// parseMonitorLogsPaginationParams parses page and limit for monitor logs with max limit clamping.
// Deprecated: Use ParsePaginationParams with MonitorLogsPaginationConfig() instead.
func parseMonitorLogsPaginationParams(c *gin.Context) (int, int, error) {
	return ParsePaginationParams(c, MonitorLogsPaginationConfig())
}

// ParsePaginationParams parses and validates pagination parameters from query string.
// Returns page (>=1), limit (clamped to max if configured), and error if invalid.
func ParsePaginationParams(c *gin.Context, config PaginationConfig) (int, int, error) {
	page := defaultPage
	limit := config.DefaultLimit

	if pageQuery := c.Query("page"); pageQuery != "" {
		parsedPage, err := strconv.Atoi(pageQuery)
		if err != nil || parsedPage < 1 {
			return 0, 0, errors.New("invalid page query parameter")
		}
		page = parsedPage
	}

	if limitQuery := c.Query("limit"); limitQuery != "" {
		parsedLimit, err := strconv.Atoi(limitQuery)
		if err != nil || parsedLimit < 1 {
			return 0, 0, errors.New("invalid limit query parameter")
		}
		if config.MaxLimit > 0 && parsedLimit > config.MaxLimit {
			parsedLimit = config.MaxLimit
		}
		limit = parsedLimit
	}

	return page, limit, nil
}

// writePaginatedResponse writes a standardized paginated JSON response.
// Uses models.PaginatedResult for type-safe response structure.
func writePaginatedResponse[T any](c *gin.Context, items []T, total, page, limit int) {
	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	result := models.PaginatedResult[T]{
		Items:      items,
		Page:       page,
		Total:      int64(total),
		TotalPages: totalPages,
	}

	c.JSON(200, result)
}
