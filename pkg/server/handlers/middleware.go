package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

// RequestIDKey is the key used to store the request ID in the context
const RequestIDKey = "RequestID"

// RequestID adds a unique request ID to each request
func SetRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request already has an ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = xid.New().String()
		}

		// Set request ID in context and header
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// Logger logs information about each request
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		requestID := c.GetString(RequestIDKey)

		// Process request
		c.Next()

		// Log after request
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		logger.Info("Request processed",
			zap.String("request_id", requestID),
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
		)
	}
}

// Recovery recovers from panics and ensures the server stays running
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := c.GetString(RequestIDKey)

				// Log the error
				logger.Error("Request handler panicked",
					zap.String("request_id", requestID),
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
				)

				// Respond with error
				c.JSON(http.StatusInternalServerError, BuildResponse("error", "Internal server error", nil))
				c.Abort()
			}
		}()

		c.Next()
	}
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(c *gin.Context) string {
	requestID, exists := c.Get(RequestIDKey)
	if !exists {
		return ""
	}
	return requestID.(string)
}

// BuildResponse creates a standardized API response
func BuildResponse(key string, msg string, data interface{}) gin.H {
	return gin.H{
		"status":  key,
		"message": msg,
		// "data":    data,
	}
}

// ValidationError represents an error that occurs during validation
type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

// validateObjectID ensures the object ID meets the requirements
func validateObjectID(id string) error {
	if len(id) == 0 || len(id) > 32 {
		return &ValidationError{message: "object ID must be between 1 and 32 characters"}
	}

	for _, char := range id {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9')) {
			return &ValidationError{message: "object ID must contain only alphanumeric characters"}
		}
	}

	return nil
}

// HandleError processes errors and returns appropriate responses
func HandleError(c *gin.Context, err error) {
	requestID := GetRequestID(c)

	switch e := err.(type) {
	case *ValidationError:
		c.JSON(http.StatusBadRequest, BuildResponse("error", e.Error(), gin.H{
			"request_id": requestID,
		}))
	default:
		c.JSON(http.StatusInternalServerError, BuildResponse("error", "Internal server error", gin.H{
			"request_id": requestID,
		}))
	}
}

const ContextLoggerKey = "contextLogger"

func WithLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Attach the logger with request ID to the context
		requestID := c.GetString(RequestIDKey)
		reqLogger := logger.With(zap.String("request_id", requestID), zap.String("path", c.FullPath()))
		c.Set(ContextLoggerKey, reqLogger)
		c.Next()
	}
}

// func GetLogger(c *gin.Context) *zap.Logger {
// 	if logger, exists := c.Get(ContextLoggerKey); exists {
// 		if zapLogger, ok := logger.(*zap.Logger); ok {
// 			return zapLogger
// 		}
// 	}
// 	return zap.NewNop() // fallback to no-op logger
// }
