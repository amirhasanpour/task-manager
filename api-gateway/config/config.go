package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Services ServicesConfig
	JWT      JWTConfig
	Logging  LoggingConfig
	Metrics  MetricsConfig
	OTel     OTelConfig
	CORS     CORSConfig
	Swagger  SwaggerConfig
}

type ServerConfig struct {
	Port                   int
	Host                   string
	GracefulShutdownTimeout time.Duration
}

type ServicesConfig struct {
	User ServiceConfig
	Todo ServiceConfig
}

type ServiceConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
}

type JWTConfig struct {
	Secret        string
	TokenLifetime time.Duration
}

type LoggingConfig struct {
	Level           string
	Encoding        string
	OutputPaths     []string
	ErrorOutputPaths []string
}

type MetricsConfig struct {
	Port int
}

type OTelConfig struct {
	Endpoint    string
	ServiceName string
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

type SwaggerConfig struct {
	Enabled bool
	Path    string
	APIPath string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Set default values
	setDefaults()

	// Read environment variables
	viper.AutomaticEnv()

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file not found, using defaults and environment variables")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.graceful_shutdown_timeout", "10s")

	viper.SetDefault("services.user.host", "localhost")
	viper.SetDefault("services.user.port", 50051)
	viper.SetDefault("services.user.timeout", "5s")

	viper.SetDefault("services.todo.host", "localhost")
	viper.SetDefault("services.todo.port", 50052)
	viper.SetDefault("services.todo.timeout", "5s")

	viper.SetDefault("jwt.secret", "your-super-secret-jwt-key-change-in-production")
	viper.SetDefault("jwt.token_lifetime", "24h")

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.encoding", "json")
	viper.SetDefault("logging.output_paths", []string{"stdout"})
	viper.SetDefault("logging.error_output_paths", []string{"stderr"})

	viper.SetDefault("metrics.port", 9091)

	viper.SetDefault("otel.endpoint", "http://localhost:4317")
	viper.SetDefault("otel.service_name", "api-gateway")

	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"Origin", "Content-Type", "Authorization", "Accept"})
	viper.SetDefault("cors.allow_credentials", true)
	viper.SetDefault("cors.max_age", "12h")

	viper.SetDefault("swagger.enabled", true)
	viper.SetDefault("swagger.path", "/swagger/*")
	viper.SetDefault("swagger.api_path", "/swagger/api.json")
}