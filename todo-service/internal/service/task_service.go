package service

import (
	"slices"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amirhasanpour/task-manager/todo-service/internal/cache"
	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type TaskService interface {
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*model.Task, error)
	GetTask(ctx context.Context, id string) (*model.Task, error)
	GetTaskByUser(ctx context.Context, id, userID string) (*model.Task, error)
	UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*model.Task, error)
	DeleteTask(ctx context.Context, id string) error
	DeleteTaskByUser(ctx context.Context, id, userID string) error
	ListTasks(ctx context.Context, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error)
	ListTasksByUser(ctx context.Context, userID string, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error)
}

type taskService struct {
	repo       repository.TaskRepository
	cache      cache.TaskCache
	metrics    *MetricsCollector
	logger     *zap.Logger
	tracer     trace.Tracer
}

type CreateTaskRequest struct {
	UserID      string
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
}

type UpdateTaskRequest struct {
	ID          string
	UserID      string
	Title       *string
	Description *string
	Status      *string
	Priority    *string
	DueDate     *time.Time
}

func NewTaskService(repo repository.TaskRepository, cache cache.TaskCache, metrics *MetricsCollector) TaskService {
	return &taskService{
		repo:    repo,
		cache:   cache,
		metrics: metrics,
		logger:  zap.L().Named("task_service"),
		tracer:  otel.Tracer("task-service"),
	}
}

func (s *taskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*model.Task, error) {
	ctx, span := s.tracer.Start(ctx, "TaskService.CreateTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.UserID),
		attribute.String("task.title", req.Title),
	)

	s.logger.Debug("Creating task", 
		zap.String("user_id", req.UserID),
		zap.String("title", req.Title),
	)

	// Validate input
	if err := s.validateCreateTaskRequest(req); err != nil {
		s.logger.Warn("Invalid create task request", zap.Error(err))
		s.metrics.IncrementValidationErrors()
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create task model
	task := &model.Task{
		UserID:      req.UserID,
		Title:       req.Title,
		Description: req.Description,
		Status:      model.StatusTodo,
		Priority:    model.PriorityMedium,
		DueDate:     req.DueDate,
	}

	// Set status if provided
	if req.Status != "" {
		task.Status = task.FromProtoStatus(req.Status)
	}

	// Set priority if provided
	if req.Priority != "" {
		task.Priority = task.FromProtoPriority(req.Priority)
	}

	// Create task in database
	createdTask, err := s.repo.Create(ctx, task)
	if err != nil {
		s.logger.Error("Failed to create task in repository", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to create task")
	}

	// Invalidate user tasks list cache (since list changed)
	if err := s.cache.InvalidateUserTasks(ctx, req.UserID); err != nil {
		s.logger.Error("Failed to invalidate user tasks cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
		// Don't fail the operation if cache invalidation fails
	}

	// Cache the newly created task
	if err := s.cache.SetTask(ctx, createdTask); err != nil {
		s.logger.Error("Failed to cache newly created task", zap.Error(err))
		s.metrics.IncrementCacheErrors()
		// Don't fail the operation if caching fails
	}

	s.logger.Info("Task created successfully", 
		zap.String("id", createdTask.ID),
		zap.String("user_id", req.UserID),
	)
	
	s.metrics.UpdateTasksCountByStatus(createdTask.ToProtoStatus(), 1)
	s.metrics.UpdateTasksCountByPriority(createdTask.ToProtoPriority(), 1)
	
	return createdTask, nil
}

func (s *taskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	ctx, span := s.tracer.Start(ctx, "TaskService.GetTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", id))

	s.logger.Debug("Getting task", zap.String("id", id))

	// Try to get from cache first
	cachedTask, err := s.cache.GetTask(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get task from cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	} else if cachedTask != nil {
		s.metrics.IncrementCacheHits()
		s.logger.Debug("Task retrieved from cache", zap.String("id", id))
		return cachedTask, nil
	}

	s.metrics.IncrementCacheMisses()

	// Get from database
	task, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get task from repository", zap.Error(err), zap.String("id", id))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to get task")
	}

	if task == nil {
		s.logger.Warn("Task not found", zap.String("id", id))
		return nil, status.Error(codes.NotFound, "task not found")
	}

	// Cache the task
	if err := s.cache.SetTask(ctx, task); err != nil {
		s.logger.Error("Failed to cache task", zap.Error(err))
		s.metrics.IncrementCacheErrors()
		// Don't fail the operation if caching fails
	}

	s.logger.Debug("Task retrieved from database", zap.String("id", id))
	return task, nil
}

func (s *taskService) GetTaskByUser(ctx context.Context, id, userID string) (*model.Task, error) {
	ctx, span := s.tracer.Start(ctx, "TaskService.GetTaskByUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("task.id", id),
		attribute.String("user.id", userID),
	)

	s.logger.Debug("Getting task by user", 
		zap.String("id", id),
		zap.String("user_id", userID),
	)

	// Try to get from cache first
	cachedTask, err := s.cache.GetTask(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get task from cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	} else if cachedTask != nil {
		if cachedTask.UserID != userID {
			s.logger.Warn("Task belongs to different user", 
				zap.String("task_id", id),
				zap.String("expected_user", userID),
				zap.String("actual_user", cachedTask.UserID),
			)
			return nil, status.Error(codes.PermissionDenied, "task not found")
		}
		s.metrics.IncrementCacheHits()
		s.logger.Debug("Task retrieved from cache by user", zap.String("id", id))
		return cachedTask, nil
	}

	s.metrics.IncrementCacheMisses()

	// Get from database
	task, err := s.repo.FindByIDAndUser(ctx, id, userID)
	if err != nil {
		s.logger.Error("Failed to get task by user from repository", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to get task")
	}

	if task == nil {
		s.logger.Warn("Task not found for user", 
			zap.String("id", id),
			zap.String("user_id", userID),
		)
		return nil, status.Error(codes.NotFound, "task not found")
	}

	// Cache the task
	if err := s.cache.SetTask(ctx, task); err != nil {
		s.logger.Error("Failed to cache task", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	s.logger.Debug("Task retrieved from database by user", zap.String("id", id))
	return task, nil
}

func (s *taskService) UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*model.Task, error) {
	ctx, span := s.tracer.Start(ctx, "TaskService.UpdateTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("task.id", req.ID),
		attribute.String("user.id", req.UserID),
	)

	s.logger.Debug("Updating task", 
		zap.String("id", req.ID),
		zap.String("user_id", req.UserID),
	)

	// Get existing task
	task, err := s.repo.FindByIDAndUser(ctx, req.ID, req.UserID)
	if err != nil {
		s.logger.Error("Failed to get task for update", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to get task")
	}

	if task == nil {
		s.logger.Warn("Task not found for update", 
			zap.String("id", req.ID),
			zap.String("user_id", req.UserID),
		)
		return nil, status.Error(codes.NotFound, "task not found")
	}

	// Track old status and priority for metrics
	oldStatus := task.ToProtoStatus()
	oldPriority := task.ToProtoPriority()

	// Update fields if provided
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = task.FromProtoStatus(*req.Status)
	}
	if req.Priority != nil {
		task.Priority = task.FromProtoPriority(*req.Priority)
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}

	// Update task in database
	updatedTask, err := s.repo.Update(ctx, task)
	if err != nil {
		s.logger.Error("Failed to update task in repository", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to update task")
	}

	// Invalidate user tasks cache
	if err := s.cache.InvalidateUserTasks(ctx, req.UserID); err != nil {
		s.logger.Error("Failed to invalidate user tasks cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	// Update cache
	if err := s.cache.SetTask(ctx, updatedTask); err != nil {
		s.logger.Error("Failed to update task in cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	// Update metrics if status or priority changed
	if req.Status != nil {
		s.metrics.UpdateTasksCountByStatus(oldStatus, -1)
		s.metrics.UpdateTasksCountByStatus(updatedTask.ToProtoStatus(), 1)
	}
	if req.Priority != nil {
		s.metrics.UpdateTasksCountByPriority(oldPriority, -1)
		s.metrics.UpdateTasksCountByPriority(updatedTask.ToProtoPriority(), 1)
	}

	s.logger.Info("Task updated successfully", zap.String("id", req.ID))
	return updatedTask, nil
}

func (s *taskService) DeleteTask(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "TaskService.DeleteTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", id))

	s.logger.Debug("Deleting task", zap.String("id", id))

	// First get the task to know the user ID for cache invalidation
	task, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get task for deletion", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return status.Error(codes.Internal, "failed to delete task")
	}

	if task == nil {
		s.logger.Warn("Task not found for deletion", zap.String("id", id))
		return status.Error(codes.NotFound, "task not found")
	}

	// Delete from database
	err = s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "task not found")
		}
		s.logger.Error("Failed to delete task from repository", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return status.Error(codes.Internal, "failed to delete task")
	}

	// Delete from cache
	if err := s.cache.DeleteTask(ctx, id); err != nil {
		s.logger.Error("Failed to delete task from cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	// Invalidate user tasks cache
	if err := s.cache.InvalidateUserTasks(ctx, task.UserID); err != nil {
		s.logger.Error("Failed to invalidate user tasks cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	// Update metrics
	s.metrics.UpdateTasksCountByStatus(task.ToProtoStatus(), -1)
	s.metrics.UpdateTasksCountByPriority(task.ToProtoPriority(), -1)

	s.logger.Info("Task deleted successfully", zap.String("id", id))
	return nil
}

func (s *taskService) DeleteTaskByUser(ctx context.Context, id, userID string) error {
	ctx, span := s.tracer.Start(ctx, "TaskService.DeleteTaskByUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("task.id", id),
		attribute.String("user.id", userID),
	)

	s.logger.Debug("Deleting task by user", 
		zap.String("id", id),
		zap.String("user_id", userID),
	)

	// First get the task to update metrics
	task, err := s.repo.FindByIDAndUser(ctx, id, userID)
	if err != nil {
		s.logger.Error("Failed to get task for deletion", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return status.Error(codes.Internal, "failed to delete task")
	}

	if task == nil {
		s.logger.Warn("Task not found for deletion by user", 
			zap.String("id", id),
			zap.String("user_id", userID),
		)
		return status.Error(codes.NotFound, "task not found")
	}

	// Delete from database
	err = s.repo.DeleteByUser(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "task not found")
		}
		s.logger.Error("Failed to delete task by user from repository", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return status.Error(codes.Internal, "failed to delete task")
	}

	// Delete from cache
	if err := s.cache.DeleteTask(ctx, id); err != nil {
		s.logger.Error("Failed to delete task from cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	// Invalidate user tasks cache
	if err := s.cache.InvalidateUserTasks(ctx, userID); err != nil {
		s.logger.Error("Failed to invalidate user tasks cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	// Update metrics
	s.metrics.UpdateTasksCountByStatus(task.ToProtoStatus(), -1)
	s.metrics.UpdateTasksCountByPriority(task.ToProtoPriority(), -1)

	s.logger.Info("Task deleted successfully by user", zap.String("id", id))
	return nil
}

func (s *taskService) ListTasks(ctx context.Context, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	ctx, span := s.tracer.Start(ctx, "TaskService.ListTasks")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	s.logger.Debug("Listing tasks", 
		zap.Int("page", page),
		zap.Int("page_size", pageSize),
	)

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Generate cache key
	cacheKey := s.generateCacheKey("list", filter, page, pageSize)

	// Try to get from cache
	cachedTasks, cachedTotal, err := s.cache.GetTasksList(ctx, cacheKey)
	if err != nil {
		s.logger.Error("Failed to get tasks list from cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	} else if cachedTasks != nil {
		s.metrics.IncrementCacheHits()
		s.logger.Debug("Tasks list retrieved from cache", 
			zap.String("key", cacheKey),
			zap.Int("count", len(cachedTasks)),
		)
		return cachedTasks, cachedTotal, nil
	}

	s.metrics.IncrementCacheMisses()

	// Get from database
	tasks, total, err := s.repo.List(ctx, filter, page, pageSize)
	if err != nil {
		s.logger.Error("Failed to list tasks from repository", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return nil, 0, status.Error(codes.Internal, "failed to list tasks")
	}

	// Cache the results
	if err := s.cache.SetTasksList(ctx, cacheKey, tasks, total); err != nil {
		s.logger.Error("Failed to cache tasks list", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	s.logger.Debug("Tasks listed successfully", 
		zap.Int("count", len(tasks)),
		zap.Int64("total", total),
	)
	return tasks, total, nil
}

func (s *taskService) ListTasksByUser(ctx context.Context, userID string, filter *repository.TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	ctx, span := s.tracer.Start(ctx, "TaskService.ListTasksByUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", userID),
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	s.logger.Debug("Listing tasks by user", 
		zap.String("user_id", userID),
		zap.Int("page", page),
		zap.Int("page_size", pageSize),
	)

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Generate cache key
	cacheKey := s.generateUserCacheKey(userID, "list", filter, page, pageSize)

	// Try to get from cache
	cachedTasks, cachedTotal, err := s.cache.GetTasksList(ctx, cacheKey)
	if err != nil {
		s.logger.Error("Failed to get user tasks list from cache", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	} else if cachedTasks != nil {
		s.metrics.IncrementCacheHits()
		s.logger.Debug("User tasks list retrieved from cache", 
			zap.String("key", cacheKey),
			zap.Int("count", len(cachedTasks)),
		)
		return cachedTasks, cachedTotal, nil
	}

	s.metrics.IncrementCacheMisses()

	// Get from database
	tasks, total, err := s.repo.ListByUser(ctx, userID, filter, page, pageSize)
	if err != nil {
		s.logger.Error("Failed to list tasks by user from repository", zap.Error(err))
		s.metrics.IncrementDatabaseErrors()
		span.RecordError(err)
		return nil, 0, status.Error(codes.Internal, "failed to list tasks")
	}

	// Cache the results
	if err := s.cache.SetTasksList(ctx, cacheKey, tasks, total); err != nil {
		s.logger.Error("Failed to cache user tasks list", zap.Error(err))
		s.metrics.IncrementCacheErrors()
	}

	s.logger.Debug("Tasks listed by user successfully", 
		zap.String("user_id", userID),
		zap.Int("count", len(tasks)),
		zap.Int64("total", total),
	)
	return tasks, total, nil
}

func (s *taskService) validateCreateTaskRequest(req *CreateTaskRequest) error {
	if req.UserID == "" {
		return errors.New("user_id is required")
	}
	if req.Title == "" {
		return errors.New("title is required")
	}
	if len(req.Title) > 255 {
		return errors.New("title must be less than 255 characters")
	}
	if req.Status != "" {
		validStatuses := []string{"TODO", "IN_PROGRESS", "DONE", "ARCHIVED"}
		if !contains(validStatuses, strings.ToUpper(req.Status)) {
			return fmt.Errorf("status must be one of: %v", validStatuses)
		}
	}
	if req.Priority != "" {
		validPriorities := []string{"LOW", "MEDIUM", "HIGH", "URGENT"}
		if !contains(validPriorities, strings.ToUpper(req.Priority)) {
			return fmt.Errorf("priority must be one of: %v", validPriorities)
		}
	}
	return nil
}

func (s *taskService) generateCacheKey(prefix string, filter *repository.TaskFilter, page, pageSize int) string {
	var parts []string
	parts = append(parts, prefix)
	
	if filter != nil {
		if filter.Status != nil && *filter.Status != "" {
			parts = append(parts, fmt.Sprintf("status:%s", *filter.Status))
		}
		if filter.Priority != nil && *filter.Priority != "" {
			parts = append(parts, fmt.Sprintf("priority:%s", *filter.Priority))
		}
		if filter.UserID != nil && *filter.UserID != "" {
			parts = append(parts, fmt.Sprintf("user:%s", *filter.UserID))
		}
		if filter.SortBy != "" {
			sortDir := "asc"
			if filter.SortDesc {
				sortDir = "desc"
			}
			parts = append(parts, fmt.Sprintf("sort:%s:%s", filter.SortBy, sortDir))
		}
	}
	
	parts = append(parts, fmt.Sprintf("page:%d", page))
	parts = append(parts, fmt.Sprintf("size:%d", pageSize))
	
	return strings.Join(parts, ":")
}

func (s *taskService) generateUserCacheKey(userID, prefix string, filter *repository.TaskFilter, page, pageSize int) string {
	var parts []string
	parts = append(parts, prefix, fmt.Sprintf("user:%s", userID))
	
	if filter != nil {
		if filter.Status != nil && *filter.Status != "" {
			parts = append(parts, fmt.Sprintf("status:%s", *filter.Status))
		}
		if filter.Priority != nil && *filter.Priority != "" {
			parts = append(parts, fmt.Sprintf("priority:%s", *filter.Priority))
		}
		if filter.SortBy != "" {
			sortDir := "asc"
			if filter.SortDesc {
				sortDir = "desc"
			}
			parts = append(parts, fmt.Sprintf("sort:%s:%s", filter.SortBy, sortDir))
		}
	}
	
	parts = append(parts, fmt.Sprintf("page:%d", page))
	parts = append(parts, fmt.Sprintf("size:%d", pageSize))
	
	return strings.Join(parts, ":")
}

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}