package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/models"
	"github.com/sirupsen/logrus"
)

type PostgresStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

// function makes a connection to the postgres database
func NewPostgresStorage(connectionString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	logger := logrus.New()

	return &PostgresStorage{
		db:     db,
		logger: logger,
	}, nil
}

// closes the database connection
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// stores a single log entry in the database
func (s *PostgresStorage) InsertLog(log *models.LogEntry) error {
	query := `
        INSERT INTO logs (timestamp, source, level, message, service, fields, raw_message, created_at, user_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
    `

	// Convert fields map to JSON
	var fieldsJSON []byte
	var err error
	if log.Fields != nil {
		fieldsJSON, err = json.Marshal(log.Fields)
		if err != nil {
			return fmt.Errorf("failed to marshal fields: %w", err)
		}
	}

	err = s.db.QueryRow(
		query,
		log.Timestamp,
		log.Source,
		log.Level,
		log.Message,
		log.Service,
		fieldsJSON,
		log.RawMessage,
		log.CreatedAt,
		log.UserID,
	).Scan(&log.ID)

	if err != nil {
		s.logger.WithError(err).Error("Failed to insert log")
		return fmt.Errorf("failed to insert log: %w", err)
	}

	return nil
}

// stores multiple log entries in a single transaction
func (s *PostgresStorage) InsertLogs(logs []*models.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
        INSERT INTO logs (timestamp, source, level, message, service, fields, raw_message, created_at, user_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, log := range logs {
		var fieldsJSON []byte
		if log.Fields != nil {
			fieldsJSON, err = json.Marshal(log.Fields)
			if err != nil {
				return fmt.Errorf("failed to marshal fields: %w", err)
			}
		}

		_, err = stmt.Exec(
			log.Timestamp,
			log.Source,
			log.Level,
			log.Message,
			log.Service,
			fieldsJSON,
			log.RawMessage,
			log.CreatedAt,
			log.UserID,
		)
		if err != nil {
			s.logger.WithError(err).Error("Failed to execute insert")
			return fmt.Errorf("failed to insert log: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Infof("Successfully inserted %d logs", len(logs))
	return nil
}

// retrieves recent log entries (for testing)
func (s *PostgresStorage) GetRecentLogs(limit int) ([]*models.LogEntry, error) {
	query := `
        SELECT id, timestamp, source, level, message, service, fields, raw_message, created_at
        FROM logs
        ORDER BY timestamp DESC
        LIMIT $1
    `

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.LogEntry
	for rows.Next() {
		log := &models.LogEntry{}
		var fieldsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Source,
			&log.Level,
			&log.Message,
			&log.Service,
			&fieldsJSON,
			&log.RawMessage,
			&log.CreatedAt,
		)
		if err != nil {
			s.logger.WithError(err).Error("Failed to scan log row")
			continue
		}

		// Unmarshal fields JSON
		if len(fieldsJSON) > 0 {
			err = json.Unmarshal(fieldsJSON, &log.Fields)
			if err != nil {
				s.logger.WithError(err).Error("Failed to unmarshal fields")
			}
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetDB exposes the database connection for auth storage
func (s *PostgresStorage) GetDB() *sql.DB {
	return s.db
}

// GetRecentLogsByUser retrieves recent log entries for a specific user
func (s *PostgresStorage) GetRecentLogsByUser(userID int, limit int) ([]*models.LogEntry, error) {
	query := `
        SELECT id, timestamp, source, level, message, service, fields, raw_message, created_at, user_id
        FROM logs
        WHERE user_id = $1
        ORDER BY timestamp DESC
        LIMIT $2
    `

	rows, err := s.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.LogEntry
	for rows.Next() {
		log := &models.LogEntry{}
		var fieldsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Source,
			&log.Level,
			&log.Message,
			&log.Service,
			&fieldsJSON,
			&log.RawMessage,
			&log.CreatedAt,
			&log.UserID,
		)
		if err != nil {
			s.logger.WithError(err).Error("Failed to scan log row")
			continue
		}

		// Unmarshal fields JSON
		if len(fieldsJSON) > 0 {
			err = json.Unmarshal(fieldsJSON, &log.Fields)
			if err != nil {
				s.logger.WithError(err).Error("Failed to unmarshal fields")
			}
		}

		logs = append(logs, log)
	}

	return logs, nil
}
