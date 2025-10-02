package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/config"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/models"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/storage"
	"github.com/sirupsen/logrus"
)

type ProcessorService struct {
	storage     *storage.PostgresStorage
	redisClient *storage.RedisClient
	logger      *logrus.Logger
	config      *config.Config
}

// creates a new processor service
func NewProcessorService(cfg *config.Config) (*ProcessorService, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Connect to PostgreSQL
	pgStorage, err := storage.NewPostgresStorage(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Connect to Redis
	redisClient, err := storage.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	return &ProcessorService{
		storage:     pgStorage,
		redisClient: redisClient,
		logger:      logger,
		config:      cfg,
	}, nil
}

func (s *ProcessorService) Close() error {
	if err := s.storage.Close(); err != nil {
		s.logger.WithError(err).Error("Failed to close database")
	}
	if err := s.redisClient.Close(); err != nil {
		s.logger.WithError(err).Error("Failed to close Redis")
	}
	return nil
}

// processLog handles a single log entry
func (s *ProcessorService) processLog(log *models.LogEntry) error {
	// Store in PostgreSQL
	if err := s.storage.InsertLog(log); err != nil {
		return fmt.Errorf("failed to store log in database: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"log_id":  log.ID,
		"user_id": log.UserID,
		"level":   log.Level,
		"source":  log.Source,
	}).Debug("Log processed and stored")

	return nil
}

// begins processing logs from Redis Stream
func (s *ProcessorService) Start(ctx context.Context) error {
	consumerGroup := "log-processors"
	consumerName := fmt.Sprintf("processor-%d", os.Getpid())

	s.logger.WithFields(logrus.Fields{
		"consumer_group": consumerGroup,
		"consumer_name":  consumerName,
	}).Info("Starting log processor")

	// Start consuming from Redis Stream
	return s.redisClient.ConsumeLogStream(ctx, consumerGroup, consumerName, s.processLog)
}

// loads configuration, initializes the processor service, runs the log process in the background
// waits for an interrupt signal, cancels the context to stop log consumption, waits briefly to allow cleanup, exits
func main() {
	cfg := config.Load()

	processor, err := NewProcessorService(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create processor service")
	}
	defer processor.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start processor in goroutine
	go func() {
		if err := processor.Start(ctx); err != nil && err != context.Canceled {
			processor.logger.WithError(err).Error("Processor stopped with error")
		}
	}()

	processor.logger.Info("Processor service started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	processor.logger.Info("Shutting down processor service...")
	cancel() // Cancel context to stop consumer

	// Give some time for graceful shutdown
	time.Sleep(2 * time.Second)
	processor.logger.Info("Processor service stopped")
}
