package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) (*model.Task, error)
	FindByID(ctx context.Context, id string) (*model.Task, error)
	FindByIDAndUser(ctx context.Context, id, userID string) (*model.Task, error)
	Update(ctx context.Context, task *model.Task) (*model.Task, error)
	Delete(ctx context.Context, id string) error
	DeleteByUser(ctx context.Context, id, userID string) error
	List(ctx context.Context, filter *TaskFilter, page, pageSize int) ([]*model.Task, int64, error)
	ListByUser(ctx context.Context, userID string, filter *TaskFilter, page, pageSize int) ([]*model.Task, int64, error)
}

type TaskFilter struct {
	Status   *string
	Priority *string
	UserID   *string
	SortBy   string
	SortDesc bool
}

type taskRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{
		db:     db,
		logger: zap.L().Named("task_repository"),
	}
}

func (r *taskRepository) Create(ctx context.Context, task *model.Task) (*model.Task, error) {
	r.logger.Debug("Creating new task", 
		zap.String("user_id", task.UserID),
		zap.String("title", task.Title),
	)

	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		r.logger.Error("Failed to create task", zap.Error(err))
		return nil, err
	}

	r.logger.Info("Task created successfully", 
		zap.String("id", task.ID),
		zap.String("user_id", task.UserID),
	)
	return task, nil
}

func (r *taskRepository) FindByID(ctx context.Context, id string) (*model.Task, error) {
	r.logger.Debug("Finding task by ID", zap.String("id", id))

	var task model.Task
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debug("Task not found", zap.String("id", id))
			return nil, nil
		}
		r.logger.Error("Failed to find task by ID", zap.Error(err), zap.String("id", id))
		return nil, err
	}

	return &task, nil
}

func (r *taskRepository) FindByIDAndUser(ctx context.Context, id, userID string) (*model.Task, error) {
	r.logger.Debug("Finding task by ID and user", 
		zap.String("id", id),
		zap.String("user_id", userID),
	)

	var task model.Task
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debug("Task not found for user", 
				zap.String("id", id),
				zap.String("user_id", userID),
			)
			return nil, nil
		}
		r.logger.Error("Failed to find task by ID and user", zap.Error(err))
		return nil, err
	}

	return &task, nil
}

func (r *taskRepository) Update(ctx context.Context, task *model.Task) (*model.Task, error) {
	r.logger.Debug("Updating task", zap.String("id", task.ID))

	result := r.db.WithContext(ctx).Save(task)
	if result.Error != nil {
		r.logger.Error("Failed to update task", 
			zap.Error(result.Error),
			zap.String("id", task.ID),
		)
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("No rows affected when updating task", zap.String("id", task.ID))
		return nil, errors.New("no task found to update")
	}

	r.logger.Info("Task updated successfully", zap.String("id", task.ID))
	return task, nil
}

func (r *taskRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("Deleting task", zap.String("id", id))

	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Task{})
	if result.Error != nil {
		r.logger.Error("Failed to delete task", 
			zap.Error(result.Error),
			zap.String("id", id),
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("No rows affected when deleting task", zap.String("id", id))
		return errors.New("no task found to delete")
	}

	r.logger.Info("Task deleted successfully", zap.String("id", id))
	return nil
}

func (r *taskRepository) DeleteByUser(ctx context.Context, id, userID string) error {
	r.logger.Debug("Deleting task by user", 
		zap.String("id", id),
		zap.String("user_id", userID),
	)

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.Task{})
	
	if result.Error != nil {
		r.logger.Error("Failed to delete task by user", zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("No rows affected when deleting task by user", 
			zap.String("id", id),
			zap.String("user_id", userID),
		)
		return errors.New("no task found to delete")
	}

	r.logger.Info("Task deleted successfully by user", zap.String("id", id))
	return nil
}

func (r *taskRepository) List(ctx context.Context, filter *TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	r.logger.Debug("Listing tasks", 
		zap.Int("page", page),
		zap.Int("pageSize", pageSize),
	)

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&model.Task{})

	// Apply filters
	if filter != nil {
		if filter.Status != nil && *filter.Status != "" {
			query = query.Where("status = ?", *filter.Status)
		}
		if filter.Priority != nil && *filter.Priority != "" {
			query = query.Where("priority = ?", *filter.Priority)
		}
		if filter.UserID != nil && *filter.UserID != "" {
			query = query.Where("user_id = ?", *filter.UserID)
		}
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error("Failed to count tasks", zap.Error(err))
		return nil, 0, err
	}

	// Apply sorting
	query = applySorting(query, filter)

	// Get paginated results
	var tasks []*model.Task
	if err := query.Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		r.logger.Error("Failed to list tasks", zap.Error(err))
		return nil, 0, err
	}

	r.logger.Debug("Tasks listed successfully", 
		zap.Int64("total", total),
		zap.Int("count", len(tasks)),
	)
	return tasks, total, nil
}

func (r *taskRepository) ListByUser(ctx context.Context, userID string, filter *TaskFilter, page, pageSize int) ([]*model.Task, int64, error) {
	r.logger.Debug("Listing tasks by user", 
		zap.String("user_id", userID),
		zap.Int("page", page),
		zap.Int("pageSize", pageSize),
	)

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&model.Task{}).Where("user_id = ?", userID)

	// Apply additional filters
	if filter != nil {
		if filter.Status != nil && *filter.Status != "" {
			query = query.Where("status = ?", *filter.Status)
		}
		if filter.Priority != nil && *filter.Priority != "" {
			query = query.Where("priority = ?", *filter.Priority)
		}
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error("Failed to count tasks by user", zap.Error(err))
		return nil, 0, err
	}

	// Apply sorting
	query = applySorting(query, filter)

	// Get paginated results
	var tasks []*model.Task
	if err := query.Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		r.logger.Error("Failed to list tasks by user", zap.Error(err))
		return nil, 0, err
	}

	r.logger.Debug("Tasks listed by user successfully", 
		zap.String("user_id", userID),
		zap.Int64("total", total),
		zap.Int("count", len(tasks)),
	)
	return tasks, total, nil
}

func applySorting(query *gorm.DB, filter *TaskFilter) *gorm.DB {
	if filter == nil || filter.SortBy == "" {
		// Default sorting by creation date descending
		return query.Order("created_at DESC")
	}

	order := "ASC"
	if filter.SortDesc {
		order = "DESC"
	}

	// Map sort field to database column
	sortField := mapSortField(filter.SortBy)
	return query.Order(fmt.Sprintf("%s %s", sortField, order))
}

func mapSortField(field string) string {
	switch strings.ToLower(field) {
	case "title":
		return "title"
	case "status":
		return "status"
	case "priority":
		return "priority"
	case "due_date":
		return "due_date"
	case "created_at":
		return "created_at"
	case "updated_at":
		return "updated_at"
	default:
		return "created_at"
	}
}