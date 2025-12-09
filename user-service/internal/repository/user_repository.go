package repository

import (
	"context"
	"errors"

	"github.com/amirhasanpour/task-manager/user-service/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) (*model.User, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error)
}

type userRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db:     db,
		logger: zap.L().Named("user_repository"),
	}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	r.logger.Debug("Creating new user", zap.String("email", user.Email))
	
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		r.logger.Error("Failed to create user", zap.Error(err), zap.String("email", user.Email))
		return nil, err
	}
	
	r.logger.Info("User created successfully", zap.String("id", user.ID), zap.String("email", user.Email))
	return user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	r.logger.Debug("Finding user by ID", zap.String("id", id))
	
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debug("User not found", zap.String("id", id))
			return nil, nil
		}
		r.logger.Error("Failed to find user by ID", zap.Error(err), zap.String("id", id))
		return nil, err
	}
	
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	r.logger.Debug("Finding user by email", zap.String("email", email))
	
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debug("User not found by email", zap.String("email", email))
			return nil, nil
		}
		r.logger.Error("Failed to find user by email", zap.Error(err), zap.String("email", email))
		return nil, err
	}
	
	return &user, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	r.logger.Debug("Finding user by username", zap.String("username", username))
	
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debug("User not found by username", zap.String("username", username))
			return nil, nil
		}
		r.logger.Error("Failed to find user by username", zap.Error(err), zap.String("username", username))
		return nil, err
	}
	
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) (*model.User, error) {
	r.logger.Debug("Updating user", zap.String("id", user.ID))
	
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		r.logger.Error("Failed to update user", zap.Error(result.Error), zap.String("id", user.ID))
		return nil, result.Error
	}
	
	if result.RowsAffected == 0 {
		r.logger.Warn("No rows affected when updating user", zap.String("id", user.ID))
		return nil, errors.New("no user found to update")
	}
	
	r.logger.Info("User updated successfully", zap.String("id", user.ID))
	return user, nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("Deleting user", zap.String("id", id))
	
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.User{})
	if result.Error != nil {
		r.logger.Error("Failed to delete user", zap.Error(result.Error), zap.String("id", id))
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		r.logger.Warn("No rows affected when deleting user", zap.String("id", id))
		return errors.New("no user found to delete")
	}
	
	r.logger.Info("User deleted successfully", zap.String("id", id))
	return nil
}

func (r *userRepository) List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error) {
	r.logger.Debug("Listing users", zap.Int("page", page), zap.Int("pageSize", pageSize))
	
	offset := (page - 1) * pageSize
	
	var users []*model.User
	var total int64
	
	// Get total count
	if err := r.db.WithContext(ctx).Model(&model.User{}).Count(&total).Error; err != nil {
		r.logger.Error("Failed to count users", zap.Error(err))
		return nil, 0, err
	}
	
	// Get paginated users
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		r.logger.Error("Failed to list users", zap.Error(err))
		return nil, 0, err
	}
	
	r.logger.Debug("Users listed successfully", zap.Int64("total", total), zap.Int("count", len(users)))
	return users, total, nil
}