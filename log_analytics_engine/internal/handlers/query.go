package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/models"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/storage"
	"github.com/sirupsen/logrus"
)

type QueryHandler struct {
	storage *storage.PostgresStorage
	logger  *logrus.Logger
}

func NewQueryHandler(storage *storage.PostgresStorage, logger *logrus.Logger) *QueryHandler {
	return &QueryHandler{
		storage: storage,
		logger:  logger,
	}
}

// QueryLogs handles POST /api/v1/logs/query
func (h *QueryHandler) QueryLogs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	var req models.QueryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid JSON in query request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON format",
			"details": err.Error(),
		})
		return
	}

	if err := req.Validate(); err != nil {
		h.logger.WithError(err).Warn("Query validation failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Convert query to SQL
	whereClause, args := req.ToSQL(userID.(int))

	// Get total count
	totalCount, err := h.storage.CountLogs(userID.(int), whereClause, args)
	if err != nil {
		h.logger.WithError(err).Error("Failed to count logs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute query",
		})
		return
	}

	// Execute query
	logs, err := h.storage.QueryLogs(userID.(int), whereClause, args, req.SortBy, req.SortOrder, req.Limit, req.Offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query logs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute query",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"total_count": totalCount,
		"returned":    len(logs),
		"level":       req.Level,
		"source":      req.Source,
		"service":     req.Service,
	}).Info("Query executed successfully")

	response := models.QueryResponse{
		Logs:       logs,
		TotalCount: totalCount,
		Limit:      req.Limit,
		Offset:     req.Offset,
		ExecutedAt: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}
