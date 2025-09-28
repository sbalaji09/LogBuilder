package models

import (
	"fmt"
	"strings"
	"time"
)

/*
This class handles all the information about the specific logs:
defines the log and incoming request structs
valides whether or not an incoming log as all the fields required
converts an incoming log to an actual log entry
*/

// single log entry
type LogEntry struct {
	ID         int64             `json:"id" db:"id"`
	Timestamp  time.Time         `json:"timestamp" db:"timestamp"`
	Source     string            `json:"source" db:"source"`
	Level      string            `json:"level" db:"level"`
	Message    string            `json:"message" db:"message"`
	Service    string            `json:"service" db:"service"`
	Fields     map[string]string `json:"fields,omitempty" db:"fields"`
	RawMessage string            `json:"raw_message,omitempty" db:"raw_message"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
}

// incoming log data
type IngestRequest struct {
	Timestamp *time.Time        `json:"timestamp,omitempty"`
	Source    string            `json:"source" binding:"required"`
	Level     string            `json:"level" binding:"required"`
	Message   string            `json:"message" binding:"required"`
	Service   string            `json:"service,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// multiple logs at once
type BatchIngestRequest struct {
	Logs []IngestRequest `json:"logs" binding:"required"`
}

// checks if the log entry has required fields
func (req *IngestRequest) Validate() error {
	if req.Source == "" {
		return fmt.Errorf("source is required")
	}
	if req.Level == "" {
		return fmt.Errorf("level is required")
	}
	if req.Message == "" {
		return fmt.Errorf("message is required")
	}

	// Validate log level
	validLevels := map[string]bool{
		"DEBUG": true, "INFO": true, "WARN": true,
		"ERROR": true, "FATAL": true,
	}
	if !validLevels[strings.ToUpper(req.Level)] {
		return fmt.Errorf("invalid log level: %s", req.Level)
	}

	return nil
}

// converts IngestRequest to LogEntry
func (req *IngestRequest) ToLogEntry() *LogEntry {
	timestamp := time.Now()
	if req.Timestamp != nil {
		timestamp = *req.Timestamp
	}

	return &LogEntry{
		Timestamp:  timestamp,
		Source:     req.Source,
		Level:      strings.ToUpper(req.Level),
		Message:    req.Message,
		Service:    req.Service,
		Fields:     req.Fields,
		RawMessage: "", // Will be filled when we add parsing
		CreatedAt:  time.Now(),
	}
}
