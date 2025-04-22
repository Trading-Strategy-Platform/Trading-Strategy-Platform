package utils

import "strings"

// NormalizeSortDirection normalizes sort direction to "ASC" or "DESC"
func NormalizeSortDirection(direction string) string {
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		return "DESC" // Default to descending
	}
	return direction
}
