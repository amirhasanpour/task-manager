package router

import (
	"github.com/gin-gonic/gin"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/handler"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/middleware"
	"github.com/amirhasanpour/task-manager/api-gateway/pkg/metrics"
	_ "github.com/amirhasanpour/task-manager/api-gateway/internal/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

type Config struct {
	Metrics          *metrics.Metrics
	UserHandler      *handler.UserHandler
	AuthHandler      *handler.AuthHandler
	TaskHandler      *handler.TaskHandler
	HealthHandler    *handler.HealthHandler
	LoggingMiddleware *middleware.LoggingMiddleware
	MetricsMiddleware *middleware.MetricsMiddleware
	AuthMiddleware   *middleware.AuthMiddleware
	CORSConfig       middleware.CORSConfig
	SwaggerEnabled   bool
	SwaggerPath      string
}

func NewRouter(cfg Config) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	
	router := gin.New()
	
	// Recovery middleware
	router.Use(gin.Recovery())
	
	// CORS middleware
	router.Use(middleware.CORSMiddleware(cfg.CORSConfig))
	
	// Logging middleware
	router.Use(cfg.LoggingMiddleware.Handler())
	
	// Metrics middleware
	router.Use(cfg.MetricsMiddleware.Handler())
	
	// Public routes
	public := router.Group("/api/v1")
	{
		// Health check
		public.GET("/health", cfg.HealthHandler.Health)
		
		// Auth routes
		auth := public.Group("/auth")
		{
			auth.POST("/register", cfg.AuthHandler.Register)
			auth.POST("/login", cfg.AuthHandler.Login)
			auth.POST("/validate", cfg.AuthHandler.ValidateToken)
		}
	}
	
	// Protected routes (require authentication)
	protected := router.Group("/api/v1")
	protected.Use(cfg.AuthMiddleware.Handler())
	{
		// User routes
		users := protected.Group("/users")
		{
			users.GET("", cfg.UserHandler.ListUsers)
			users.POST("", cfg.UserHandler.CreateUser)
			users.GET("/:id", cfg.UserHandler.GetUser)
			users.PUT("/:id", cfg.UserHandler.UpdateUser)
			users.DELETE("/:id", cfg.UserHandler.DeleteUser)
			users.GET("/me", cfg.UserHandler.GetCurrentUser)
			users.PUT("/me", cfg.UserHandler.UpdateCurrentUser)
		}
		
		// Task routes
		tasks := protected.Group("/tasks")
		{
			tasks.GET("", cfg.TaskHandler.ListTasks)
			tasks.POST("", cfg.TaskHandler.CreateTask)
			tasks.GET("/:id", cfg.TaskHandler.GetTask)
			tasks.PUT("/:id", cfg.TaskHandler.UpdateTask)
			tasks.DELETE("/:id", cfg.TaskHandler.DeleteTask)
			
			// User-specific task routes
			tasks.GET("/me", cfg.TaskHandler.ListMyTasks)
		}
	}
	
	// Swagger documentation
	if cfg.SwaggerEnabled {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		zap.L().Info("Swagger documentation enabled", zap.String("path", cfg.SwaggerPath))
	}
	
	// Metrics endpoint (separate from API)
	router.GET("/metrics", func(c *gin.Context) {
		// This will be handled by prometheus client library
		// We just need to ensure the route exists
		c.JSON(200, gin.H{"message": "Metrics are available at /metrics endpoint"})
	})
	
	return router
}