package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amirhasanpour/task-manager/api-gateway/config"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/client"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/handler"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/middleware"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/router"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/tracing"
	"github.com/amirhasanpour/task-manager/api-gateway/pkg/logger"
	"github.com/amirhasanpour/task-manager/api-gateway/pkg/metrics"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	loggerConfig := logger.Config{
		Level:            cfg.Logging.Level,
		Encoding:         cfg.Logging.Encoding,
		OutputPaths:      cfg.Logging.OutputPaths,
		ErrorOutputPaths: cfg.Logging.ErrorOutputPaths,
	}

	if err := logger.InitLogger(loggerConfig); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	log := logger.GetLogger()
	log.Info("Starting API Gateway",
		zap.String("version", "1.0.0"),
		zap.String("environment", os.Getenv("APP_ENV")),
	)

	// Initialize tracing
	ctx := context.Background()
	shutdownTracer, err := tracing.InitTracerProvider(ctx, tracing.Config{
		Endpoint:    cfg.OTel.Endpoint,
		ServiceName: cfg.OTel.ServiceName,
	})
	if err != nil {
		log.Error("Failed to initialize tracing", zap.Error(err))
	} else {
		defer func() {
			if err := shutdownTracer(ctx); err != nil {
				log.Error("Failed to shutdown tracer", zap.Error(err))
			}
		}()
	}

	// Initialize metrics
	metricsCollector := metrics.NewMetrics("api_gateway")
	metricsCollector.StartMetricsServer(fmt.Sprintf("%d", cfg.Metrics.Port))

	// Initialize gRPC clients
	userClient, err := client.NewUserClient(client.UserConfig{
		Host:    cfg.Services.User.Host,
		Port:    cfg.Services.User.Port,
		Timeout: cfg.Services.User.Timeout,
	})
	if err != nil {
		log.Error("Failed to create user client", zap.Error(err))
		os.Exit(1)
	}
	defer userClient.Close()

	todoClient, err := client.NewTodoClient(client.TodoConfig{
		Host:    cfg.Services.Todo.Host,
		Port:    cfg.Services.Todo.Port,
		Timeout: cfg.Services.Todo.Timeout,
	})
	if err != nil {
		log.Error("Failed to create todo client", zap.Error(err))
		os.Exit(1)
	}
	defer todoClient.Close()

	// Initialize handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(userClient)
	userHandler := handler.NewUserHandler(userClient)
	taskHandler := handler.NewTaskHandler(todoClient)

	// Initialize middleware
	loggingMiddleware := middleware.NewLoggingMiddleware()
	metricsMiddleware := middleware.NewMetricsMiddleware(metricsCollector)
	authMiddleware := middleware.NewAuthMiddleware(userClient, cfg.JWT.Secret)

	// Create router
	ginRouter := router.NewRouter(router.Config{
		Metrics:           metricsCollector,
		UserHandler:       userHandler,
		AuthHandler:       authHandler,
		TaskHandler:       taskHandler,
		HealthHandler:     healthHandler,
		LoggingMiddleware: loggingMiddleware,
		MetricsMiddleware: metricsMiddleware,
		AuthMiddleware:    authMiddleware,
		CORSConfig: middleware.CORSConfig{
			AllowedOrigins:   cfg.CORS.AllowedOrigins,
			AllowedMethods:   cfg.CORS.AllowedMethods,
			AllowedHeaders:   cfg.CORS.AllowedHeaders,
			AllowCredentials: cfg.CORS.AllowCredentials,
			MaxAge:           cfg.CORS.MaxAge,
		},
		SwaggerEnabled: cfg.Swagger.Enabled,
		SwaggerPath:    cfg.Swagger.Path,
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      ginRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Starting HTTP server", 
			zap.String("address", server.Addr),
			zap.Int("port", cfg.Server.Port),
			zap.Bool("swagger_enabled", cfg.Swagger.Enabled),
		)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start HTTP server", zap.Error(err))
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server shutdown complete")
}