// server/server.go
package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	objectstorage "github.com/singhmeghna79/homework-object-storage/pkg/internals/objectStorage"
	"github.com/singhmeghna79/homework-object-storage/pkg/server/handlers"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server encapsulates the HTTP server and its dependencies
type App struct {
	port        string
	logger      *zap.Logger
	storageType string
}

// New creates a new server instance
func New(port string, storageType string, logger *zap.Logger) *App {
	return &App{
		port:        port,
		logger:      logger,
		storageType: storageType,
	}
}

// Run starts the server and blocks until shutdown is complete
func (s *App) Run() {
	// Set Gin mode to release for production
	gin.SetMode(gin.ReleaseMode)

	// Initialize router with routes and middleware
	router := s.setupRouter()

	// Initialize HTTP server
	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		s.logger.Info("Starting server", zap.String("port", s.port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-quit
	s.logger.Info("Shutdown signal received", zap.String("signal", sig.String()))

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	s.logger.Info("Shutting down server")
	if err := server.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", zap.Error(err))
	}

	s.logger.Info("Server exited gracefully")
}

// setupRouter configures the Gin router with all routes and middleware
func (s *App) setupRouter() *gin.Engine {
	router := gin.New()

	// Add middlewares
	router.Use(handlers.SetRequestID())
	router.Use(handlers.WithLogger(s.logger))
	router.Use(handlers.Logger(s.logger))
	router.Use(handlers.Recovery(s.logger))

	objectStorageFactory := objectstorage.NewObjectStorageFactory()
	storageService := objectStorageFactory.GetObjectStorage(s.storageType, s.logger)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, handlers.BuildResponse("health", "OK", nil))
	})

	// API group with version
	v1 := router.Group("/api/v1")
	{
		// Objects API
		objects := v1.Group("/object")
		{
			objects.GET("/:id", handlers.HandleGetObject(storageService))
			objects.PUT("/:id", handlers.HandlePutObject(storageService))
		}
	}

	return router
}
