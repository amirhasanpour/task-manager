package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Config struct {
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

type RedisClient struct {
	client    *redis.Client
	logger    *zap.Logger
	cacheTTL  time.Duration
}

func NewRedisClient(cfg Config) (*RedisClient, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger := zap.L().Named("redis")
	logger.Info("Successfully connected to Redis", 
		zap.String("address", addr),
		zap.Int("db", cfg.DB),
	)

	return &RedisClient{
		client:   rdb,
		logger:   logger,
		cacheTTL: cfg.CacheTTL,
	}, nil
}

func (r *RedisClient) Set(ctx context.Context, key string, value any) error {
	r.logger.Debug("Setting cache key", zap.String("key", key))
	
	err := r.client.Set(ctx, key, value, r.cacheTTL).Err()
	if err != nil {
		r.logger.Error("Failed to set cache key", zap.Error(err), zap.String("key", key))
		return err
	}
	
	return nil
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	r.logger.Debug("Getting cache key", zap.String("key", key))
	
	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			r.logger.Debug("Cache miss", zap.String("key", key))
			return "", nil
		}
		r.logger.Error("Failed to get cache key", zap.Error(err), zap.String("key", key))
		return "", err
	}
	
	r.logger.Debug("Cache hit", zap.String("key", key))
	return value, nil
}

func (r *RedisClient) Delete(ctx context.Context, key string) error {
	r.logger.Debug("Deleting cache key", zap.String("key", key))
	
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		r.logger.Error("Failed to delete cache key", zap.Error(err), zap.String("key", key))
		return err
	}
	
	r.logger.Debug("Cache key deleted", zap.String("key", key))
	return nil
}

func (r *RedisClient) DeletePattern(ctx context.Context, pattern string) error {
	r.logger.Debug("Deleting cache pattern", zap.String("pattern", pattern))
	
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		if err := r.Delete(ctx, key); err != nil {
			return err
		}
	}
	
	if err := iter.Err(); err != nil {
		r.logger.Error("Failed to scan cache keys", zap.Error(err))
		return err
	}
	
	r.logger.Debug("Cache pattern deleted", zap.String("pattern", pattern))
	return nil
}

func (r *RedisClient) Close() error {
	r.logger.Info("Closing Redis connection")
	return r.client.Close()
}