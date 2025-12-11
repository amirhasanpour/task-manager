package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amirhasanpour/task-manager/todo-service/config"
	"github.com/amirhasanpour/task-manager/todo-service/internal/cache"
	"github.com/amirhasanpour/task-manager/todo-service/internal/handler"
	"github.com/amirhasanpour/task-manager/todo-service/internal/interceptor"
	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
	"github.com/amirhasanpour/task-manager/todo-service/internal/service"
	"github.com/amirhasanpour/task-manager/todo-service/internal/tracing"
	"github.com/amirhasanpour/task-manager/todo-service/pkg/db"
	"github.com/amirhasanpour/task-manager/todo-service/pkg/logger"
	"github.com/amirhasanpour/task-manager/todo-service/pkg/metrics"
	"github.com/amirhasanpour/task-manager/todo-service/pkg/redis"
	pb "github.com/amirhasanpour/task-manager/todo-service/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
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
	log.Info("Starting Todo Service",
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
	metricsCollector := metrics.NewMetrics("todo_service")
	metricsCollector.StartMetricsServer(fmt.Sprintf("%d", cfg.Metrics.Port))

	// Initialize database connection
	dbConfig := db.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Name:            cfg.Database.Name,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	database, err := db.NewPostgresConnection(dbConfig)
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	// Run database migrations
	if err := db.Migrate(database, &model.Task{}); err != nil {
		log.Error("Failed to migrate database", zap.Error(err))
		os.Exit(1)
	}

	// Initialize Redis client
	redisConfig := redis.Config{
		Host:         cfg.Redis.Host,
		Port:         cfg.Redis.Port,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
		CacheTTL:     cfg.Redis.CacheTTL,
	}

	redisClient, err := redis.NewRedisClient(redisConfig)
	if err != nil {
		log.Error("Failed to connect to Redis", zap.Error(err))
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize repository
	taskRepo := repository.NewTaskRepository(database)

	// Initialize cache
	taskCache := cache.NewTaskCache(redisClient)

	// Initialize service metrics
	serviceMetrics := service.NewMetricsCollector(
		func(count int) { metricsCollector.UpdateTasksCount(count) },
		func(status string, count int) { metricsCollector.UpdateTasksCountByStatus(status, count) },
		func(priority string, count int) { metricsCollector.UpdateTasksCountByPriority(priority, count) },
		func() { metricsCollector.IncrementCacheHits() },
		func() { metricsCollector.IncrementCacheMisses() },
		func() { metricsCollector.IncrementDatabaseErrors() },
		func() { metricsCollector.IncrementCacheErrors() },
		func() { metricsCollector.IncrementValidationErrors() },
	)

	// Initialize service
	taskService := service.NewTaskService(taskRepo, taskCache, serviceMetrics)

	// Initialize handler
	taskHandler := handler.NewTaskHandler(taskService)

	// Initialize interceptors
	metricsInterceptor := interceptor.NewMetricsInterceptor(metricsCollector)
	loggingInterceptor := interceptor.NewLoggingInterceptor()
	recoveryInterceptor := interceptor.NewRecoveryInterceptor()

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor.Unary(),
			loggingInterceptor.Unary(),
			metricsInterceptor.Unary(),
		),
	)

	// Register services
	pb.RegisterTodoServiceServer(grpcServer, taskHandler)
	
	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("todo-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection service (for debugging)
	reflection.Register(grpcServer)

	// Start gRPC server
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Error("Failed to create listener", zap.Error(err))
		os.Exit(1)
	}

	log.Info("Starting gRPC server", zap.String("address", address))

	// Start server in a goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Error("Failed to serve gRPC", zap.Error(err))
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Set health status to NOT_SERVING
	healthServer.SetServingStatus("todo-service", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Graceful stop gRPC server
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop with timeout
	select {
	case <-stopped:
		log.Info("Server stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Warn("Force stopping server after timeout")
		grpcServer.Stop()
	}

	log.Info("Server shutdown complete")
}