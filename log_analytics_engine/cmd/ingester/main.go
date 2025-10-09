package main

import (
	"context"
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
	storage      *storage.PostgresStorage
	redisClient  *storage.RedisClient
	authStorage  *storage.AuthStorage
	authHandler  *handlers.AuthHandler
	queryHandler *handlers.QueryHandler
	jwtService   *auth.JWTService
	logger       *logrus.Logger
	config       *config.Config
}

func NewIngestionService(cfg *config.Config) (*IngestionService, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Connect to database (still needed for auth)
	pgStorage, err := storage.NewPostgresStorage(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Connect to Redis
	redisClient, err := storage.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	// Create auth storage
	authStorage := storage.NewAuthStorage(pgStorage.GetDB())

	// Create JWT service
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer)

	// Create auth handler
	authHandler := handlers.NewAuthHandler(authStorage, redisClient, jwtService, logger)

	// Create query handler
	queryHandler := handlers.NewQueryHandler(pgStorage, logger)

	return &IngestionService{
		storage:      pgStorage,
		redisClient:  redisClient,
		authStorage:  authStorage,
		authHandler:  authHandler,
		queryHandler: queryHandler,
		jwtService:   jwtService,
		logger:       logger,
		config:       cfg,
	}, nil
}

func (s *IngestionService) Close() error {
	if err := s.storage.Close(); err != nil {
		s.logger.WithError(err).Error("Failed to close database")
	}
	if err := s.redisClient.Close(); err != nil {
		s.logger.WithError(err).Error("Failed to close Redis")
	}
	return nil
}

// Health check endpoint
func (s *IngestionService) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check Redis connection
	redisHealthy := true
	if err := s.redisClient.GetClient().Ping(ctx).Err(); err != nil {
		redisHealthy = false
		s.logger.WithError(err).Warn("Redis health check failed")
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !redisHealthy {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":        status,
		"timestamp":     time.Now(),
		"service":       "log-ingestion",
		"redis_healthy": redisHealthy,
	})
}

// Ingest a single log entry (now requires API key)
func (s *IngestionService) IngestLog(c *gin.Context) {
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
	logEntry.UserID = userID.(int)

	// Publish to Redis Stream instead of direct database insert
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.redisClient.PublishLog(ctx, logEntry); err != nil {
		s.logger.WithError(err).Error("Failed to publish log to Redis")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to queue log for processing",
		})
		return
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"source":  logEntry.Source,
		"level":   logEntry.Level,
		"service": logEntry.Service,
	}).Info("Log queued successfully")

	c.JSON(http.StatusAccepted, gin.H{
		"status":    "queued",
		"timestamp": logEntry.Timestamp,
		"message":   "Log accepted and queued for processing",
	})
}

// Ingest multiple log entries (now requires API key)
func (s *IngestionService) IngestBatch(c *gin.Context) {
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
		entry.UserID = userID.(int)
		logEntries = append(logEntries, entry)
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Validation failed for some logs",
			"validation_errors": validationErrors,
		})
		return
	}

	// Publish batch to Redis Stream
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.redisClient.PublishLogs(ctx, logEntries); err != nil {
		s.logger.WithError(err).Error("Failed to publish batch logs to Redis")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to queue logs for processing",
		})
		return
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"count":   len(logEntries),
	}).Info("Batch logs queued successfully")

	c.JSON(http.StatusAccepted, gin.H{
		"status":      "queued",
		"logs_queued": len(logEntries),
		"timestamp":   time.Now(),
		"message":     "Logs accepted and queued for processing",
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

func (s *IngestionService) GetStreamStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	info, err := s.redisClient.GetStreamInfo(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get stream info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get stream status",
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

func setupRouter(service *IngestionService) *gin.Engine {
	if service.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Static("/", "./static")

	// CORS middleware
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

	// Public routes
	api := router.Group("/api/v1")
	{
		api.GET("/health", service.HealthCheck)
		api.POST("/auth/register", service.authHandler.Register)
		api.POST("/auth/login", service.authHandler.Login)
	}

	// Protected routes (JWT)
	protected := router.Group("/api/v1")
	protected.Use(service.authHandler.JWTAuthMiddleware())
	{
		protected.POST("/api-keys", service.authHandler.CreateAPIKey)
		protected.GET("/api-keys", service.authHandler.GetAPIKeys)
		protected.DELETE("/api-keys/:id", service.authHandler.DeleteAPIKey)
		protected.GET("/stream/status", service.GetStreamStatus)
	}

	// Log query routes (JWT or API key)
	logsQuery := router.Group("/api/v1/logs")
	logsQuery.Use(service.authHandler.JWTOrAPIKeyAuthMiddleware())
	{
		logsQuery.GET("/recent", service.GetRecentLogs)
		logsQuery.POST("/query", service.queryHandler.QueryLogs)
	}

	// Log ingestion routes (API key only for security)
	logsIngest := router.Group("/api/v1/logs")
	logsIngest.Use(service.authHandler.APIKeyAuthMiddleware())
	{
		logsIngest.POST("/ingest", service.IngestLog)
		logsIngest.POST("/batch", service.IngestBatch)
	}

	return router
}

func main() {
	cfg := config.Load()

	service, err := NewIngestionService(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create ingestion service")
	}
	defer service.Close()

	router := setupRouter(service)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		service.logger.Infof("Starting ingestion service on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			service.logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	service.logger.Info("Shutting down ingestion service...")
}
