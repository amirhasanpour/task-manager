package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/pkg/redis"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type TaskCache interface {
	GetTask(ctx context.Context, id string) (*model.Task, error)
	SetTask(ctx context.Context, task *model.Task) error
	DeleteTask(ctx context.Context, id string) error
	GetTasksList(ctx context.Context, key string) ([]*model.Task, int64, error)
	SetTasksList(ctx context.Context, key string, tasks []*model.Task, total int64) error
	DeleteTasksList(ctx context.Context, pattern string) error
	InvalidateUserTasks(ctx context.Context, userID string) error
}

type taskCache struct {
	redisClient *redis.RedisClient
	logger      *zap.Logger
	tracer      trace.Tracer
}

func NewTaskCache(redisClient *redis.RedisClient) TaskCache {
	return &taskCache{
		redisClient: redisClient,
		logger:      zap.L().Named("task_cache"),
		tracer:      otel.Tracer("task-cache"),
	}
}

func (c *taskCache) GetTask(ctx context.Context, id string) (*model.Task, error) {
	ctx, span := c.tracer.Start(ctx, "TaskCache.GetTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", id))

	cacheKey := c.taskKey(id)
	c.logger.Debug("Getting task from cache", zap.String("key", cacheKey))

	data, err := c.redisClient.Get(ctx, cacheKey)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if data == "" {
		c.logger.Debug("Task cache miss", zap.String("key", cacheKey))
		return nil, nil
	}

	var task model.Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		c.logger.Error("Failed to unmarshal cached task", 
			zap.Error(err),
			zap.String("key", cacheKey),
		)
		span.RecordError(err)
		return nil, err
	}

	c.logger.Debug("Task cache hit", zap.String("key", cacheKey))
	return &task, nil
}

func (c *taskCache) SetTask(ctx context.Context, task *model.Task) error {
	ctx, span := c.tracer.Start(ctx, "TaskCache.SetTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", task.ID))

	cacheKey := c.taskKey(task.ID)
	c.logger.Debug("Setting task in cache", zap.String("key", cacheKey))

	data, err := json.Marshal(task)
	if err != nil {
		c.logger.Error("Failed to marshal task for cache", 
			zap.Error(err),
			zap.String("task_id", task.ID),
		)
		span.RecordError(err)
		return err
	}

	if err := c.redisClient.Set(ctx, cacheKey, data); err != nil {
		span.RecordError(err)
		return err
	}

	c.logger.Debug("Task cached successfully", zap.String("key", cacheKey))
	return nil
}

func (c *taskCache) DeleteTask(ctx context.Context, id string) error {
	ctx, span := c.tracer.Start(ctx, "TaskCache.DeleteTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", id))

	cacheKey := c.taskKey(id)
	c.logger.Debug("Deleting task from cache", zap.String("key", cacheKey))

	if err := c.redisClient.Delete(ctx, cacheKey); err != nil {
		span.RecordError(err)
		return err
	}

	c.logger.Debug("Task cache deleted", zap.String("key", cacheKey))
	return nil
}

func (c *taskCache) GetTasksList(ctx context.Context, key string) ([]*model.Task, int64, error) {
	ctx, span := c.tracer.Start(ctx, "TaskCache.GetTasksList")
	defer span.End()

	span.SetAttributes(attribute.String("cache.key", key))

	c.logger.Debug("Getting tasks list from cache", zap.String("key", key))

	data, err := c.redisClient.Get(ctx, key)
	if err != nil {
		span.RecordError(err)
		return nil, 0, err
	}

	if data == "" {
		c.logger.Debug("Tasks list cache miss", zap.String("key", key))
		return nil, 0, nil
	}

	var cacheData struct {
		Tasks []*model.Task `json:"tasks"`
		Total int64         `json:"total"`
	}

	if err := json.Unmarshal([]byte(data), &cacheData); err != nil {
		c.logger.Error("Failed to unmarshal cached tasks list", 
			zap.Error(err),
			zap.String("key", key),
		)
		span.RecordError(err)
		return nil, 0, err
	}

	c.logger.Debug("Tasks list cache hit", 
		zap.String("key", key),
		zap.Int("task_count", len(cacheData.Tasks)),
	)
	return cacheData.Tasks, cacheData.Total, nil
}

func (c *taskCache) SetTasksList(ctx context.Context, key string, tasks []*model.Task, total int64) error {
	ctx, span := c.tracer.Start(ctx, "TaskCache.SetTasksList")
	defer span.End()

	span.SetAttributes(
		attribute.String("cache.key", key),
		attribute.Int("task_count", len(tasks)),
		attribute.Int64("total", total),
	)

	c.logger.Debug("Setting tasks list in cache", 
		zap.String("key", key),
		zap.Int("task_count", len(tasks)),
	)

	cacheData := struct {
		Tasks []*model.Task `json:"tasks"`
		Total int64         `json:"total"`
	}{
		Tasks: tasks,
		Total: total,
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		c.logger.Error("Failed to marshal tasks list for cache", 
			zap.Error(err),
			zap.String("key", key),
		)
		span.RecordError(err)
		return err
	}

	if err := c.redisClient.Set(ctx, key, data); err != nil {
		span.RecordError(err)
		return err
	}

	c.logger.Debug("Tasks list cached successfully", 
		zap.String("key", key),
		zap.Int("task_count", len(tasks)),
	)
	return nil
}

func (c *taskCache) DeleteTasksList(ctx context.Context, pattern string) error {
	ctx, span := c.tracer.Start(ctx, "TaskCache.DeleteTasksList")
	defer span.End()

	span.SetAttributes(attribute.String("cache.pattern", pattern))

	c.logger.Debug("Deleting tasks list from cache", zap.String("pattern", pattern))

	if err := c.redisClient.DeletePattern(ctx, pattern); err != nil {
		span.RecordError(err)
		return err
	}

	c.logger.Debug("Tasks list cache deleted", zap.String("pattern", pattern))
	return nil
}

func (c *taskCache) InvalidateUserTasks(ctx context.Context, userID string) error {
	ctx, span := c.tracer.Start(ctx, "TaskCache.InvalidateUserTasks")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", userID))

	// Pattern for all user-related cache keys
	pattern := fmt.Sprintf("tasks:user:%s:*", userID)
	
	c.logger.Debug("Invalidating user tasks cache", 
		zap.String("user_id", userID),
		zap.String("pattern", pattern),
	)

	if err := c.DeleteTasksList(ctx, pattern); err != nil {
		span.RecordError(err)
		return err
	}

	c.logger.Debug("User tasks cache invalidated", zap.String("user_id", userID))
	return nil
}

func (c *taskCache) taskKey(id string) string {
	return fmt.Sprintf("task:%s", id)
}

func (c *taskCache) tasksListKey(filterKey string) string {
	return fmt.Sprintf("tasks:list:%s", filterKey)
}

func (c *taskCache) userTasksKey(userID, filterKey string) string {
	return fmt.Sprintf("tasks:user:%s:%s", userID, filterKey)
}