package models

type PaginatedResult[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}
