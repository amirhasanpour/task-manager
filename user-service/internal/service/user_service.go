package service

import (
	"context"
	"errors"

	"github.com/amirhasanpour/task-manager/user-service/internal/auth"
	"github.com/amirhasanpour/task-manager/user-service/internal/model"
	"github.com/amirhasanpour/task-manager/user-service/internal/repository"
	"github.com/amirhasanpour/task-manager/user-service/pkg/hash"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) (*model.User, error)
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, page, pageSize int) ([]*model.User, int64, error)
	Register(ctx context.Context, req *RegisterRequest) (*model.User, string, error)
	Login(ctx context.Context, email, password string) (*model.User, string, error)
	ValidateToken(ctx context.Context, token string) (*model.User, error)
}

type userService struct {
	repo       repository.UserRepository
	jwtManager *auth.JWTManager
	logger     *zap.Logger
	tracer     trace.Tracer
}

type CreateUserRequest struct {
	Username string
	Email    string
	Password string
	FullName string
}

type UpdateUserRequest struct {
	ID       string
	Username *string
	Email    *string
	Password *string
	FullName *string
}

type RegisterRequest struct {
	Username string
	Email    string
	Password string
	FullName string
}

func NewUserService(repo repository.UserRepository, jwtManager *auth.JWTManager) UserService {
	return &userService{
		repo:       repo,
		jwtManager: jwtManager,
		logger:     zap.L().Named("user_service"),
		tracer:     otel.Tracer("user-service"),
	}
}

func (s *userService) CreateUser(ctx context.Context, req *CreateUserRequest) (*model.User, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.CreateUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("user.username", req.Username),
	)

	s.logger.Debug("Creating user", zap.String("email", req.Email))

	// Check if user with email already exists
	existingUser, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("Failed to check existing user by email", zap.Error(err))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to check existing user")
	}
	if existingUser != nil {
		s.logger.Warn("User with email already exists", zap.String("email", req.Email))
		return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
	}

	// Check if user with username already exists
	existingUser, err = s.repo.FindByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Error("Failed to check existing user by username", zap.Error(err))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to check existing user")
	}
	if existingUser != nil {
		s.logger.Warn("User with username already exists", zap.String("username", req.Username))
		return nil, status.Error(codes.AlreadyExists, "user with this username already exists")
	}

	// Hash password
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to process password")
	}

	// Create user
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
	}

	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		s.logger.Error("Failed to create user in repository", zap.Error(err))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	// Clear password before returning
	createdUser.Password = ""

	s.logger.Info("User created successfully", zap.String("id", createdUser.ID))
	span.SetAttributes(attribute.String("user.id", createdUser.ID))
	return createdUser, nil
}

func (s *userService) GetUser(ctx context.Context, id string) (*model.User, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", id))

	s.logger.Debug("Getting user", zap.String("id", id))

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get user", zap.Error(err), zap.String("id", id))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	if user == nil {
		s.logger.Warn("User not found", zap.String("id", id))
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Clear password before returning
	user.Password = ""

	s.logger.Debug("User retrieved successfully", zap.String("id", id))
	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*model.User, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.UpdateUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.ID))

	s.logger.Debug("Updating user", zap.String("id", req.ID))

	// Get existing user
	user, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("Failed to get user for update", zap.Error(err), zap.String("id", req.ID))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	if user == nil {
		s.logger.Warn("User not found for update", zap.String("id", req.ID))
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Update fields if provided
	if req.Username != nil {
		// Check if username is already taken by another user
		existingUser, err := s.repo.FindByUsername(ctx, *req.Username)
		if err != nil {
			s.logger.Error("Failed to check existing username", zap.Error(err))
			span.RecordError(err)
			return nil, status.Error(codes.Internal, "failed to check username availability")
		}
		if existingUser != nil && existingUser.ID != req.ID {
			s.logger.Warn("Username already taken", zap.String("username", *req.Username))
			return nil, status.Error(codes.AlreadyExists, "username already taken")
		}
		user.Username = *req.Username
	}

	if req.Email != nil {
		// Check if email is already taken by another user
		existingUser, err := s.repo.FindByEmail(ctx, *req.Email)
		if err != nil {
			s.logger.Error("Failed to check existing email", zap.Error(err))
			span.RecordError(err)
			return nil, status.Error(codes.Internal, "failed to check email availability")
		}
		if existingUser != nil && existingUser.ID != req.ID {
			s.logger.Warn("Email already taken", zap.String("email", *req.Email))
			return nil, status.Error(codes.AlreadyExists, "email already taken")
		}
		user.Email = *req.Email
	}

	if req.Password != nil {
		hashedPassword, err := hash.HashPassword(*req.Password)
		if err != nil {
			s.logger.Error("Failed to hash password", zap.Error(err))
			span.RecordError(err)
			return nil, status.Error(codes.Internal, "failed to process password")
		}
		user.Password = hashedPassword
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}

	// Update user
	updatedUser, err := s.repo.Update(ctx, user)
	if err != nil {
		s.logger.Error("Failed to update user", zap.Error(err), zap.String("id", req.ID))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	// Clear password before returning
	updatedUser.Password = ""

	s.logger.Info("User updated successfully", zap.String("id", req.ID))
	return updatedUser, nil
}

func (s *userService) DeleteUser(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "UserService.DeleteUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", id))

	s.logger.Debug("Deleting user", zap.String("id", id))

	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error("Failed to delete user", zap.Error(err), zap.String("id", id))
		span.RecordError(err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "user not found")
		}
		return status.Error(codes.Internal, "failed to delete user")
	}

	s.logger.Info("User deleted successfully", zap.String("id", id))
	return nil
}

func (s *userService) ListUsers(ctx context.Context, page, pageSize int) ([]*model.User, int64, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.ListUsers")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)

	s.logger.Debug("Listing users", zap.Int("page", page), zap.Int("page_size", pageSize))

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

	users, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("Failed to list users", zap.Error(err))
		span.RecordError(err)
		return nil, 0, status.Error(codes.Internal, "failed to list users")
	}

	// Clear passwords before returning
	for _, user := range users {
		user.Password = ""
	}

	s.logger.Debug("Users listed successfully", zap.Int("count", len(users)), zap.Int64("total", total))
	return users, total, nil
}

func (s *userService) Register(ctx context.Context, req *RegisterRequest) (*model.User, string, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.Register")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("user.username", req.Username),
	)

	s.logger.Debug("Registering user", zap.String("email", req.Email))

	// Check if user with email already exists
	existingUser, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("Failed to check existing user by email", zap.Error(err))
		span.RecordError(err)
		return nil, "", status.Error(codes.Internal, "failed to check existing user")
	}
	if existingUser != nil {
		s.logger.Warn("User with email already exists", zap.String("email", req.Email))
		return nil, "", status.Error(codes.AlreadyExists, "user with this email already exists")
	}

	// Check if user with username already exists
	existingUser, err = s.repo.FindByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Error("Failed to check existing user by username", zap.Error(err))
		span.RecordError(err)
		return nil, "", status.Error(codes.Internal, "failed to check existing user")
	}
	if existingUser != nil {
		s.logger.Warn("User with username already exists", zap.String("username", req.Username))
		return nil, "", status.Error(codes.AlreadyExists, "user with this username already exists")
	}

	// Hash password
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		span.RecordError(err)
		return nil, "", status.Error(codes.Internal, "failed to process password")
	}

	// Create user
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
	}

	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		s.logger.Error("Failed to create user in repository", zap.Error(err))
		span.RecordError(err)
		return nil, "", status.Error(codes.Internal, "failed to create user")
	}

	// Generate JWT token
	token, err := s.jwtManager.Generate(createdUser)
	if err != nil {
		s.logger.Error("Failed to generate token", zap.Error(err))
		span.RecordError(err)
		return nil, "", status.Error(codes.Internal, "failed to generate token")
	}

	// Clear password before returning
	createdUser.Password = ""

	s.logger.Info("User registered successfully", zap.String("id", createdUser.ID))
	span.SetAttributes(attribute.String("user.id", createdUser.ID))
	return createdUser, token, nil
}

func (s *userService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.Login")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", email))

	s.logger.Debug("User login attempt", zap.String("email", email))

	// Find user by email
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to find user by email", zap.Error(err), zap.String("email", email))
		span.RecordError(err)
		return nil, "", status.Error(codes.Internal, "failed to find user")
	}

	if user == nil {
		s.logger.Warn("User not found for login", zap.String("email", email))
		return nil, "", status.Error(codes.NotFound, "invalid credentials")
	}

	// Check password
	if !hash.CheckPasswordHash(password, user.Password) {
		s.logger.Warn("Invalid password attempt", zap.String("email", email))
		return nil, "", status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Generate JWT token
	token, err := s.jwtManager.Generate(user)
	if err != nil {
		s.logger.Error("Failed to generate token", zap.Error(err))
		span.RecordError(err)
		return nil, "", status.Error(codes.Internal, "failed to generate token")
	}

	// Clear password before returning
	user.Password = ""

	s.logger.Info("User logged in successfully", zap.String("id", user.ID))
	span.SetAttributes(attribute.String("user.id", user.ID))
	return user, token, nil
}

func (s *userService) ValidateToken(ctx context.Context, token string) (*model.User, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.ValidateToken")
	defer span.End()

	s.logger.Debug("Validating token")

	valid, claims := s.jwtManager.Validate(token)
	if !valid || claims == nil {
		s.logger.Warn("Invalid token")
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	// Get user from database
	user, err := s.repo.FindByID(ctx, claims.UserID)
	if err != nil {
		s.logger.Error("Failed to find user by ID", zap.Error(err), zap.String("user_id", claims.UserID))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "failed to validate user")
	}

	if user == nil {
		s.logger.Warn("User not found for token validation", zap.String("user_id", claims.UserID))
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Clear password before returning
	user.Password = ""

	s.logger.Debug("Token validated successfully", zap.String("user_id", user.ID))
	span.SetAttributes(attribute.String("user.id", user.ID))
	return user, nil
}