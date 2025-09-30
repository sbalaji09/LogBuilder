package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/auth"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/config"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/handlers"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/models"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/storage"
	"github.com/sirupsen/logrus"
)

type IngestionService struct {
	storage     *storage.PostgresStorage
	authStorage *storage.AuthStorage
	authHandler *handlers.AuthHandler
	jwtService  *auth.JWTService
	logger      *logrus.Logger
	config      *config.Config
}

func NewIngestionService(cfg *config.Config) (*IngestionService, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Connect to database
	pgStorage, err := storage.NewPostgresStorage(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Create auth storage
	authStorage := storage.NewAuthStorage(pgStorage.GetDB())

	// Create JWT service
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer)

	// Create auth handler
	authHandler := handlers.NewAuthHandler(authStorage, jwtService, logger)

	return &IngestionService{
		storage:     pgStorage,
		authStorage: authStorage,
		authHandler: authHandler,
		jwtService:  jwtService,
		logger:      logger,
		config:      cfg,
	}, nil
}

func (s *IngestionService) Close() error {
	return s.storage.Close()
}

// Health check endpoint
func (s *IngestionService) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "log-ingestion",
	})
}

// Ingest a single log entry (now requires API key)
func (s *IngestionService) IngestLog(c *gin.Context) {
	// Get user_id from context (set by API key middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		s.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

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
	logEntry.UserID = userID.(int) // Associate log with user

	// Store in database
	if err := s.storage.InsertLog(logEntry); err != nil {
		s.logger.WithError(err).Error("Failed to store log")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store log",
		})
		return
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
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

// Ingest multiple log entries (now requires API key)
func (s *IngestionService) IngestBatch(c *gin.Context) {
	// Get user_id from context
	userID, exists := c.Get("user_id")
	if !exists {
		s.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

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
		entry := logReq.ToLogEntry()
		entry.UserID = userID.(int) // Associate each log with user
		logEntries = append(logEntries, entry)
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

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"count":   len(logEntries),
	}).Info("Batch logs ingested successfully")

	c.JSON(http.StatusCreated, gin.H{
		"status":       "accepted",
		"logs_created": len(logEntries),
		"timestamp":    time.Now(),
	})
}

// Get recent logs for the authenticated user
func (s *IngestionService) GetRecentLogs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	logs, err := s.storage.GetRecentLogsByUser(userID.(int), 50)
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
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Public routes (no authentication required)
	public := router.Group("/api/v1")
	{
		public.GET("/health", service.HealthCheck)
		public.POST("/auth/register", service.authHandler.Register)
		public.POST("/auth/login", service.authHandler.Login)
	}

	// Protected routes (require JWT token for web interface)
	protected := router.Group("/api/v1")
	protected.Use(service.authHandler.JWTAuthMiddleware())
	{
		protected.POST("/api-keys", service.authHandler.CreateAPIKey)
		protected.GET("/api-keys", service.authHandler.GetAPIKeys)
		protected.DELETE("/api-keys/:id", service.authHandler.DeleteAPIKey)
	}

	// Log ingestion routes (require API key)
	logs := router.Group("/api/v1/logs")
	logs.Use(service.authHandler.APIKeyAuthMiddleware())
	{
		logs.POST("/ingest", service.IngestLog)
		logs.POST("/batch", service.IngestBatch)
		logs.GET("/recent", service.GetRecentLogs)
	}

	return router
}

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
