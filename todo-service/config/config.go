package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Logging  LoggingConfig
	Metrics  MetricsConfig
	OTel     OTelConfig
}

type ServerConfig struct {
	Port int
	Host string
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	CacheTTL     time.Duration
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
	viper.SetDefault("server.port", 50052)
	viper.SetDefault("server.host", "0.0.0.0")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "taskmanager")
	viper.SetDefault("database.password", "taskmanager123")
	viper.SetDefault("database.name", "taskmanager")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_conns", 2)
	viper.SetDefault("redis.max_retries", 3)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")
	viper.SetDefault("redis.cache_ttl", "5m")

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.encoding", "json")
	viper.SetDefault("logging.output_paths", []string{"stdout"})
	viper.SetDefault("logging.error_output_paths", []string{"stderr"})

	viper.SetDefault("metrics.port", 9093)

	viper.SetDefault("otel.endpoint", "http://localhost:4317")
	viper.SetDefault("otel.service_name", "todo-service")
}