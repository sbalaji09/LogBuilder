package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/models"
)

type AuthStorage struct {
	db *sql.DB
}

func NewAuthStorage(db *sql.DB) *AuthStorage {
	return &AuthStorage{db: db}
}

// User management
func (s *AuthStorage) CreateUser(user *models.User) error {
	query := `
        INSERT INTO users (username, email, password_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `

	now := time.Now()
	err := s.db.QueryRow(
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		now,
		now,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.CreatedAt = now
	user.UpdatedAt = now
	return nil
}

func (s *AuthStorage) GetUserByUsername(username string) (*models.User, error) {
	query := `
        SELECT id, username, email, password_hash, created_at, updated_at, is_active
        FROM users
        WHERE username = $1 AND is_active = true
    `

	user := &models.User{}
	err := s.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *AuthStorage) GetUserByEmail(email string) (*models.User, error) {
	query := `
        SELECT id, username, email, password_hash, created_at, updated_at, is_active
        FROM users
        WHERE email = $1 AND is_active = true
    `

	user := &models.User{}
	err := s.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *AuthStorage) GetUserByID(id int) (*models.User, error) {
	query := `
        SELECT id, username, email, password_hash, created_at, updated_at, is_active
        FROM users
        WHERE id = $1 AND is_active = true
    `

	user := &models.User{}
	err := s.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// API Key management
func (s *AuthStorage) CreateAPIKey(userID int, name string) (*models.APIKey, error) {
	// Generate a secure API key
	apiKeyBytes := make([]byte, 32)
	if _, err := rand.Read(apiKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}
	apiKeyString := "lak_" + hex.EncodeToString(apiKeyBytes) // lak = log analytics key

	query := `
        INSERT INTO api_keys (user_id, api_key, name, created_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `

	apiKey := &models.APIKey{
		UserID:    userID,
		APIKey:    apiKeyString,
		Name:      name,
		CreatedAt: time.Now(),
		IsActive:  true,
	}

	err := s.db.QueryRow(
		query,
		apiKey.UserID,
		apiKey.APIKey,
		apiKey.Name,
		apiKey.CreatedAt,
	).Scan(&apiKey.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return apiKey, nil
}

func (s *AuthStorage) ValidateAPIKey(apiKey string) (*models.User, error) {
	query := `
        SELECT u.id, u.username, u.email, u.password_hash, u.created_at, u.updated_at, u.is_active,
               k.id as key_id
        FROM users u
        JOIN api_keys k ON u.id = k.user_id
        WHERE k.api_key = $1 AND k.is_active = true AND u.is_active = true
    `

	user := &models.User{}
	var keyID int

	err := s.db.QueryRow(query, apiKey).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
		&keyID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	// Update last_used_at timestamp
	go s.updateAPIKeyLastUsed(keyID)

	return user, nil
}

func (s *AuthStorage) updateAPIKeyLastUsed(keyID int) {
	query := `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`
	s.db.Exec(query, time.Now(), keyID)
}

func (s *AuthStorage) GetUserAPIKeys(userID int) ([]*models.APIKey, error) {
	query := `
        SELECT id, user_id, api_key, name, created_at, last_used_at, is_active
        FROM api_keys
        WHERE user_id = $1
        ORDER BY created_at DESC
    `

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*models.APIKey
	for rows.Next() {
		key := &models.APIKey{}
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.APIKey,
			&key.Name,
			&key.CreatedAt,
			&key.LastUsedAt,
			&key.IsActive,
		)
		if err != nil {
			continue // Skip invalid rows
		}
		apiKeys = append(apiKeys, key)
	}

	return apiKeys, nil
}

func (s *AuthStorage) DeactivateAPIKey(keyID int, userID int) error {
	query := `
        UPDATE api_keys 
        SET is_active = false 
        WHERE id = $1 AND user_id = $2
    `

	result, err := s.db.Exec(query, keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found or not owned by user")
	}

	return nil
}

func (s *AuthStorage) DeleteAPIKey(keyID int, userID int) error {
	query := `
	DELETE FROM api_keys 
	WHERE id = $1 AND user_id = $2
	`

	result, err := s.db.Exec(query, keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found or not owned by user")
	}

	return nil
}
