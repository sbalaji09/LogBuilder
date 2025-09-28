package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/config"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/models"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/storage"
	"github.com/sirupsen/logrus"
)

/*
main entry point and API server for the log ingestion service, using Gin
*/

type IngestionService struct {
	storage *storage.PostgresStorage
	logger  *logrus.Logger
	config  *config.Config
}

// creates an IngestionService, sets up logging with a specified log level, and connects to the database
func NewIngestionService(cfg *config.Config) (*IngestionService, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Connect to database
	storage, err := storage.NewPostgresStorage(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &IngestionService{
		storage: storage,
		logger:  logger,
		config:  cfg,
	}, nil
}

// closes the database connection when the service shuts down
func (s *IngestionService) Close() error {
	return s.storage.Close()
}

// function responds with JSON showing service status and timestamp for health monitoring
func (s *IngestionService) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "log-ingestion",
	})
}

// accepts a single log as JSON, validates it, converts it to the standard log model, and stores it within the database
func (s *IngestionService) IngestLog(c *gin.Context) {
	var req models.IngestRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.WithError(err).Warn("Invalid JSON in request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON format",
			"details": err.Error(),
		})
		return
	}

	// Validate the request
	if err := req.Validate(); err != nil {
		s.logger.WithError(err).Warn("Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Convert to log entry
	logEntry := req.ToLogEntry()

	// Store in database
	if err := s.storage.InsertLog(logEntry); err != nil {
		s.logger.WithError(err).Error("Failed to store log")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store log",
		})
		return
	}

	s.logger.WithFields(logrus.Fields{
		"source":  logEntry.Source,
		"level":   logEntry.Level,
		"service": logEntry.Service,
	}).Info("Log ingested successfully")

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"log_id":    logEntry.ID,
		"timestamp": logEntry.Timestamp,
	})
}

// accepts an array of log entries, validates each, reports individual validation errors,
// and inserts all valid logs in a single transaction
func (s *IngestionService) IngestBatch(c *gin.Context) {
	var req models.BatchIngestRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.WithError(err).Warn("Invalid JSON in batch request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON format",
			"details": err.Error(),
		})
		return
	}

	if len(req.Logs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No logs provided in batch",
		})
		return
	}

	if len(req.Logs) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Batch size too large (max 1000 logs)",
		})
		return
	}

	var logEntries []*models.LogEntry
	var validationErrors []string

	for i, logReq := range req.Logs {
		if err := logReq.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Log %d: %s", i, err.Error()))
			continue
		}
		logEntries = append(logEntries, logReq.ToLogEntry())
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Validation failed for some logs",
			"validation_errors": validationErrors,
		})
		return
	}

	// Store all logs in a transaction
	if err := s.storage.InsertLogs(logEntries); err != nil {
		s.logger.WithError(err).Error("Failed to store batch logs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store logs",
		})
		return
	}

	s.logger.WithField("count", len(logEntries)).Info("Batch logs ingested successfully")

	c.JSON(http.StatusCreated, gin.H{
		"status":       "accepted",
		"logs_created": len(logEntries),
		"timestamp":    time.Now(),
	})
}

// retrieves recent logs for inspection or testing purposes, responds with the logs and their count
func (s *IngestionService) GetRecentLogs(c *gin.Context) {
	logs, err := s.storage.GetRecentLogs(50)
	if err != nil {
		s.logger.WithError(err).Error("Failed to retrieve logs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve logs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}

func setupRouter(service *IngestionService) *gin.Engine {
	if service.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware for development
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", service.HealthCheck)
		v1.POST("/logs/ingest", service.IngestLog)
		v1.POST("/logs/batch", service.IngestBatch)
		v1.GET("/logs/recent", service.GetRecentLogs) // For testing
	}

	return router
}

// loads configuration, initializes services, sets up routing, starts the HTTP server, and handles shutdown signals
func main() {
	// Load configuration
	cfg := config.Load()

	// Create ingestion service
	service, err := NewIngestionService(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create ingestion service")
	}
	defer service.Close()

	// Setup HTTP router
	router := setupRouter(service)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		service.logger.Infof("Starting ingestion service on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			service.logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	service.logger.Info("Shutting down ingestion service...")
}
