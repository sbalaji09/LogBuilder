package models

import (
	"fmt"
	"strings"
	"time"
)

// QueryRequest represents a query for logs with SQL-like filters
type QueryRequest struct {
	// WHERE clause filters - Single value
	Level   string `json:"level,omitempty"`   // Filter by log level (DEBUG, INFO, WARN, ERROR, FATAL)
	Source  string `json:"source,omitempty"`  // Filter by source
	Service string `json:"service,omitempty"` // Filter by service
	Message string `json:"message,omitempty"` // Search in message (substring match - CONTAINS)

	// WHERE clause filters - Multi-value (IN operator)
	Levels   []string `json:"levels,omitempty"`   // Filter by multiple log levels
	Sources  []string `json:"sources,omitempty"`  // Filter by multiple sources
	Services []string `json:"services,omitempty"` // Filter by multiple services

	// Exclusion filters (NOT)
	ExcludeLevel   string   `json:"exclude_level,omitempty"`   // Exclude a log level
	ExcludeLevels  []string `json:"exclude_levels,omitempty"`  // Exclude multiple log levels
	ExcludeSource  string   `json:"exclude_source,omitempty"`  // Exclude a source
	ExcludeSources []string `json:"exclude_sources,omitempty"` // Exclude multiple sources

	// Text search operators
	MessageContains    string `json:"message_contains,omitempty"`     // Message contains text (case-insensitive)
	MessageNotContains string `json:"message_not_contains,omitempty"` // Message does not contain text

	// Time range filters
	StartTime *time.Time `json:"start_time,omitempty"` // Filter logs after this time
	EndTime   *time.Time `json:"end_time,omitempty"`   // Filter logs before this time

	// Time range helpers (alternative to start_time/end_time)
	LastMinutes int `json:"last_minutes,omitempty"` // Logs from last N minutes
	LastHours   int `json:"last_hours,omitempty"`   // Logs from last N hours
	LastDays    int `json:"last_days,omitempty"`    // Logs from last N days

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
	validLevels := map[string]bool{
		"DEBUG": true, "INFO": true, "WARN": true,
		"ERROR": true, "FATAL": true,
	}

	// Validate log level if provided
	if q.Level != "" {
		if !validLevels[strings.ToUpper(q.Level)] {
			return fmt.Errorf("invalid log level: %s (must be DEBUG, INFO, WARN, ERROR, or FATAL)", q.Level)
		}
		q.Level = strings.ToUpper(q.Level)
	}

	// Validate multiple levels
	for i, level := range q.Levels {
		if !validLevels[strings.ToUpper(level)] {
			return fmt.Errorf("invalid log level in levels array: %s", level)
		}
		q.Levels[i] = strings.ToUpper(level)
	}

	// Validate exclude level
	if q.ExcludeLevel != "" {
		if !validLevels[strings.ToUpper(q.ExcludeLevel)] {
			return fmt.Errorf("invalid exclude_level: %s", q.ExcludeLevel)
		}
		q.ExcludeLevel = strings.ToUpper(q.ExcludeLevel)
	}

	// Validate exclude levels
	for i, level := range q.ExcludeLevels {
		if !validLevels[strings.ToUpper(level)] {
			return fmt.Errorf("invalid log level in exclude_levels: %s", level)
		}
		q.ExcludeLevels[i] = strings.ToUpper(level)
	}

	// Process time range helpers
	now := time.Now()
	if q.LastMinutes > 0 {
		startTime := now.Add(-time.Duration(q.LastMinutes) * time.Minute)
		q.StartTime = &startTime
	}
	if q.LastHours > 0 {
		startTime := now.Add(-time.Duration(q.LastHours) * time.Hour)
		q.StartTime = &startTime
	}
	if q.LastDays > 0 {
		startTime := now.Add(-time.Duration(q.LastDays) * 24 * time.Hour)
		q.StartTime = &startTime
	}

	// Validate time range
	if q.StartTime != nil && q.EndTime != nil {
		if q.StartTime.After(*q.EndTime) {
			return fmt.Errorf("start_time must be before end_time")
		}
	}

	// Handle backward compatibility: if Message is set but MessageContains is not, use Message
	if q.Message != "" && q.MessageContains == "" {
		q.MessageContains = q.Message
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

	// Single value filters
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

	// Multi-value filters (IN operator)
	if len(q.Levels) > 0 {
		placeholders := make([]string, len(q.Levels))
		for i, level := range q.Levels {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, level)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("level IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(q.Sources) > 0 {
		placeholders := make([]string, len(q.Sources))
		for i, source := range q.Sources {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, source)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("source IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(q.Services) > 0 {
		placeholders := make([]string, len(q.Services))
		for i, service := range q.Services {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, service)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("service IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Exclusion filters (NOT)
	if q.ExcludeLevel != "" {
		conditions = append(conditions, fmt.Sprintf("level != $%d", argIndex))
		args = append(args, q.ExcludeLevel)
		argIndex++
	}

	if len(q.ExcludeLevels) > 0 {
		placeholders := make([]string, len(q.ExcludeLevels))
		for i, level := range q.ExcludeLevels {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, level)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("level NOT IN (%s)", strings.Join(placeholders, ", ")))
	}

	if q.ExcludeSource != "" {
		conditions = append(conditions, fmt.Sprintf("source != $%d", argIndex))
		args = append(args, q.ExcludeSource)
		argIndex++
	}

	if len(q.ExcludeSources) > 0 {
		placeholders := make([]string, len(q.ExcludeSources))
		for i, source := range q.ExcludeSources {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, source)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("source NOT IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Text search operators
	if q.MessageContains != "" {
		conditions = append(conditions, fmt.Sprintf("message ILIKE $%d", argIndex))
		args = append(args, "%"+q.MessageContains+"%")
		argIndex++
	}

	if q.MessageNotContains != "" {
		conditions = append(conditions, fmt.Sprintf("message NOT ILIKE $%d", argIndex))
		args = append(args, "%"+q.MessageNotContains+"%")
		argIndex++
	}

	// Time range filters
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
