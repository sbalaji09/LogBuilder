package models

import (
	"fmt"
	"strings"
	"time"
)

// QueryRequest represents a query for logs with SQL-like filters
type QueryRequest struct {
	// WHERE clause filters
	Level     string    `json:"level,omitempty"`      // Filter by log level (DEBUG, INFO, WARN, ERROR, FATAL)
	Source    string    `json:"source,omitempty"`     // Filter by source
	Service   string    `json:"service,omitempty"`    // Filter by service
	Message   string    `json:"message,omitempty"`    // Search in message (substring match)
	StartTime *time.Time `json:"start_time,omitempty"` // Filter logs after this time
	EndTime   *time.Time `json:"end_time,omitempty"`   // Filter logs before this time

	// Pagination
	Limit  int `json:"limit,omitempty"`  // Number of results (default 100, max 1000)
	Offset int `json:"offset,omitempty"` // Skip N results

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // Field to sort by (default: timestamp)
	SortOrder string `json:"sort_order,omitempty"` // ASC or DESC (default: DESC)
}

// QueryResponse contains the query results
type QueryResponse struct {
	Logs       []*LogEntry `json:"logs"`
	TotalCount int         `json:"total_count"` // Total matching logs (for pagination)
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	ExecutedAt time.Time   `json:"executed_at"`
}

// Validate checks if the query parameters are valid
func (q *QueryRequest) Validate() error {
	// Validate log level if provided
	if q.Level != "" {
		validLevels := map[string]bool{
			"DEBUG": true, "INFO": true, "WARN": true,
			"ERROR": true, "FATAL": true,
		}
		if !validLevels[strings.ToUpper(q.Level)] {
			return fmt.Errorf("invalid log level: %s (must be DEBUG, INFO, WARN, ERROR, or FATAL)", q.Level)
		}
		q.Level = strings.ToUpper(q.Level)
	}

	// Validate time range
	if q.StartTime != nil && q.EndTime != nil {
		if q.StartTime.After(*q.EndTime) {
			return fmt.Errorf("start_time must be before end_time")
		}
	}

	// Set default limit
	if q.Limit <= 0 {
		q.Limit = 100
	}
	if q.Limit > 1000 {
		return fmt.Errorf("limit cannot exceed 1000")
	}

	// Validate offset
	if q.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	// Validate sort field
	if q.SortBy == "" {
		q.SortBy = "timestamp"
	}
	validSortFields := map[string]bool{
		"timestamp": true,
		"level":     true,
		"source":    true,
		"service":   true,
	}
	if !validSortFields[strings.ToLower(q.SortBy)] {
		return fmt.Errorf("invalid sort_by field: %s (must be timestamp, level, source, or service)", q.SortBy)
	}

	// Validate sort order
	if q.SortOrder == "" {
		q.SortOrder = "DESC"
	}
	q.SortOrder = strings.ToUpper(q.SortOrder)
	if q.SortOrder != "ASC" && q.SortOrder != "DESC" {
		return fmt.Errorf("invalid sort_order: %s (must be ASC or DESC)", q.SortOrder)
	}

	return nil
}

// ToSQL converts the query to SQL WHERE clauses
func (q *QueryRequest) ToSQL(userID int) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Always filter by user_id
	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	// Add filters
	if q.Level != "" {
		conditions = append(conditions, fmt.Sprintf("level = $%d", argIndex))
		args = append(args, q.Level)
		argIndex++
	}

	if q.Source != "" {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argIndex))
		args = append(args, q.Source)
		argIndex++
	}

	if q.Service != "" {
		conditions = append(conditions, fmt.Sprintf("service = $%d", argIndex))
		args = append(args, q.Service)
		argIndex++
	}

	if q.Message != "" {
		conditions = append(conditions, fmt.Sprintf("message ILIKE $%d", argIndex))
		args = append(args, "%"+q.Message+"%")
		argIndex++
	}

	if q.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, q.StartTime)
		argIndex++
	}

	if q.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, q.EndTime)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")
	return whereClause, args
}
