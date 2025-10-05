package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/auth"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/models"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/storage"
	"github.com/sirupsen/logrus"
)

type AuthHandler struct {
	authStorage *storage.AuthStorage
	redisClient *storage.RedisClient
	jwtService  *auth.JWTService
	logger      *logrus.Logger
}

// creates a new AuthHandler with JWT and logger and other dependencies
func NewAuthHandler(authStorage *storage.AuthStorage, redisClient *storage.RedisClient, jwtService *auth.JWTService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		authStorage: authStorage,
		redisClient: redisClient,
		jwtService:  jwtService,
		logger:      logger,
	}
}

// when the user registers, this generates a unique JWT token for the user after checking the username and password
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Check if username already exists
	if existingUser, _ := h.authStorage.GetUserByUsername(req.Username); existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Username already exists",
		})
		return
	}

	// Check if email already exists
	if existingUser, _ := h.authStorage.GetUserByEmail(req.Email); existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Email already exists",
		})
		return
	}

	// Create new user
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		IsActive: true,
	}

	if err := user.SetPassword(req.Password); err != nil {
		h.logger.WithError(err).Error("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	if err := h.authStorage.CreateUser(user); err != nil {
		h.logger.WithError(err).Error("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	h.logger.WithField("username", user.Username).Info("User registered successfully")

	// Remove password hash from response
	user.PasswordHash = ""

	c.JSON(http.StatusCreated, models.AuthResponse{
		User:  user,
		Token: token,
	})
}

// on login, this will generate a JWT token for the session for the user
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get user by username
	user, err := h.authStorage.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	h.logger.WithField("username", user.Username).Info("User logged in successfully")

	// Remove password hash from response
	user.PasswordHash = ""

	c.JSON(http.StatusOK, models.AuthResponse{
		User:  user,
		Token: token,
	})
}

// creates an API key for an authenticated user
func (h *AuthHandler) CreateAPIKey(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req models.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	apiKey, err := h.authStorage.CreateAPIKey(userID.(int), req.Name)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create API key",
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIKeyResponse{
		ID:        apiKey.ID,
		APIKey:    apiKey.APIKey,
		Name:      apiKey.Name,
		CreatedAt: apiKey.CreatedAt,
		IsActive:  apiKey.IsActive,
	})
}

func (h *AuthHandler) GetAPIKeys(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	apiKeys, err := h.authStorage.GetUserAPIKeys(userID.(int))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get API keys")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get API keys",
		})
		return
	}

	var response []models.APIKeyResponse
	for _, key := range apiKeys {
		// Don't return the actual API key value for security
		maskedKey := key.APIKey[:8] + "..." + key.APIKey[len(key.APIKey)-4:]
		response = append(response, models.APIKeyResponse{
			ID:        key.ID,
			APIKey:    maskedKey,
			Name:      key.Name,
			CreatedAt: key.CreatedAt,
			IsActive:  key.IsActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys": response,
		"count":    len(response),
	})
}

func (h *AuthHandler) DeleteAPIKey(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid API key ID",
		})
		return
	}

	// Get the API key string before deletion (for cache invalidation)
	apiKeys, err := h.authStorage.GetUserAPIKeys(userID.(int))
	if err == nil {
		for _, key := range apiKeys {
			if key.ID == keyID {
				// Invalidate from cache
				go func(apiKey string) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := h.redisClient.InvalidateCachedAPIKey(ctx, apiKey); err != nil {
						h.logger.WithError(err).Warn("Failed to invalidate cached API key")
					}
				}(key.APIKey)
				break
			}
		}
	}

	if err := h.authStorage.DeactivateAPIKey(keyID, userID.(int)); err != nil {
		h.logger.WithError(err).Error("Failed to delete API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete API key",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key deleted successfully",
	})
}

// extracts the JWT token from the header, validates it, and on success, stores user_id and username in Gin context for downstream handlers
func (h *AuthHandler) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format",
			})
			c.Abort()
			return
		}

		claims, err := h.jwtService.ValidateToken(tokenParts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// extracts the API key from the header and validates it through storage
func (h *AuthHandler) APIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key in Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key required",
			})
			c.Abort()
			return
		}

		// Extract API key from "Bearer <api_key>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format. Use: Bearer <api_key>",
			})
			c.Abort()
			return
		}

		apiKey := tokenParts[1]
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Try to get from Redis cache first
		userID, err := h.redisClient.GetCachedAPIKey(ctx, apiKey)
		if err == nil {
			// Cache hit - use cached user ID
			h.logger.Debug("API key validated from cache")
			c.Set("user_id", userID)
			c.Next()
			return
		}

		// Cache miss - validate from database
		h.logger.Debug("API key not in cache, validating from database")
		user, err := h.authStorage.ValidateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
			c.Abort()
			return
		}

		// Cache the API key for 15 minutes
		go func() {
			cacheCtx, cacheCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cacheCancel()
			if err := h.redisClient.CacheAPIKey(cacheCtx, apiKey, user.ID, 15*time.Minute); err != nil {
				h.logger.WithError(err).Warn("Failed to cache API key")
			}
		}()

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Next()
	}
}

// JWTOrAPIKeyAuthMiddleware accepts both JWT tokens and API keys
func (h *AuthHandler) JWTOrAPIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token/key from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// Try JWT validation first
		claims, err := h.jwtService.ValidateToken(token)
		if err == nil {
			// Valid JWT token
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Next()
			return
		}

		// JWT validation failed, try API key
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Try to get from Redis cache first
		userID, err := h.redisClient.GetCachedAPIKey(ctx, token)
		if err == nil {
			// Cache hit - use cached user ID
			h.logger.Debug("API key validated from cache")
			c.Set("user_id", userID)
			c.Next()
			return
		}

		// Cache miss - validate from database
		user, err := h.authStorage.ValidateAPIKey(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token/API key",
			})
			c.Abort()
			return
		}

		// Cache the API key for 15 minutes
		go func() {
			cacheCtx, cacheCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cacheCancel()
			if err := h.redisClient.CacheAPIKey(cacheCtx, token, user.ID, 15*time.Minute); err != nil {
				h.logger.WithError(err).Warn("Failed to cache API key")
			}
		}()

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Next()
	}
}
