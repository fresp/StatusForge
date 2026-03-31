package handlers

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage      = 1
	defaultLimit     = 20
	logsDefaultLimit = 10
	logsMaxLimit     = 100
)

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func parsePaginationParams(c *gin.Context) (int, int, error) {
	page := defaultPage
	limit := defaultLimit

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
		limit = parsedLimit
	}

	return page, limit, nil
}

func parseMonitorLogsPaginationParams(c *gin.Context) (int, int, error) {
	page := defaultPage
	limit := logsDefaultLimit

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
		if parsedLimit > logsMaxLimit {
			parsedLimit = logsMaxLimit
		}
		limit = parsedLimit
	}

	return page, limit, nil
}

func writePaginatedResponse(c *gin.Context, items any, total, page, limit int) {
	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	c.JSON(200, gin.H{
		"items":       items,
		"total":       total,
		"page":        page,
		"total_pages": totalPages,
	})
}
