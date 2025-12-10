package handler

import (
	"context"

	"github.com/amirhasanpour/task-manager/user-service/internal/model"
	"github.com/amirhasanpour/task-manager/user-service/internal/service"
	pb "github.com/amirhasanpour/task-manager/user-service/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	service service.UserService
	logger  *zap.Logger
	tracer  trace.Tracer
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  zap.L().Named("user_handler"),
		tracer:  trace.NewNoopTracerProvider().Tracer("noop"),
	}
}

func (h *UserHandler) SetTracer(tracer trace.Tracer) {
	h.tracer = tracer
}

func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.CreateUser")
	defer span.End()

	h.logger.Debug("CreateUser request received", 
		zap.String("email", req.Email),
		zap.String("username", req.Username),
	)

	serviceReq := &service.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}

	user, err := h.service.CreateUser(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		return nil, err
	}

	resp := &pb.CreateUserResponse{
		User: modelToProto(user),
	}

	h.logger.Info("CreateUser completed successfully", zap.String("user_id", user.ID))
	return resp, nil
}

func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.GetUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.Id))

	h.logger.Debug("GetUser request received", zap.String("id", req.Id))

	user, err := h.service.GetUser(ctx, req.Id)
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err), zap.String("id", req.Id))
		return nil, err
	}

	resp := &pb.GetUserResponse{
		User: modelToProto(user),
	}

	h.logger.Debug("GetUser completed successfully", zap.String("id", req.Id))
	return resp, nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.UpdateUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.Id))

	h.logger.Debug("UpdateUser request received", zap.String("id", req.Id))

	serviceReq := &service.UpdateUserRequest{
		ID: req.Id,
	}

	// Only set fields that are provided (not empty strings)
	if req.Username != "" {
		serviceReq.Username = &req.Username
	}
	if req.Email != "" {
		serviceReq.Email = &req.Email
	}
	if req.FullName != "" {
		serviceReq.FullName = &req.FullName
	}
	if req.Password != "" {
		serviceReq.Password = &req.Password
	}

	user, err := h.service.UpdateUser(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to update user", zap.Error(err), zap.String("id", req.Id))
		return nil, err
	}

	resp := &pb.UpdateUserResponse{
		User: modelToProto(user),
	}

	h.logger.Info("UpdateUser completed successfully", zap.String("id", req.Id))
	return resp, nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.DeleteUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.Id))

	h.logger.Debug("DeleteUser request received", zap.String("id", req.Id))

	err := h.service.DeleteUser(ctx, req.Id)
	if err != nil {
		h.logger.Error("Failed to delete user", zap.Error(err), zap.String("id", req.Id))
		return nil, err
	}

	resp := &pb.DeleteUserResponse{
		Success: true,
	}

	h.logger.Info("DeleteUser completed successfully", zap.String("id", req.Id))
	return resp, nil
}

func (h *UserHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.ListUsers")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", int(req.Page)),
		attribute.Int("page_size", int(req.PageSize)),
	)

	h.logger.Debug("ListUsers request received", 
		zap.Int32("page", req.Page),
		zap.Int32("page_size", req.PageSize),
	)

	page := int(req.Page)
	pageSize := int(req.PageSize)

	users, total, err := h.service.ListUsers(ctx, page, pageSize)
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		return nil, err
	}

	protoUsers := make([]*pb.User, len(users))
	for i, user := range users {
		protoUsers[i] = modelToProto(user)
	}

	resp := &pb.ListUsersResponse{
		Users:    protoUsers,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	h.logger.Debug("ListUsers completed successfully", 
		zap.Int("user_count", len(users)),
		zap.Int64("total", total),
	)
	return resp, nil
}

func (h *UserHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.Register")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("user.username", req.Username),
	)

	h.logger.Debug("Register request received", zap.String("email", req.Email))

	serviceReq := &service.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}

	user, token, err := h.service.Register(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to register user", zap.Error(err))
		return nil, err
	}

	resp := &pb.RegisterResponse{
		User:  modelToProto(user),
		Token: token,
	}

	h.logger.Info("Register completed successfully", zap.String("user_id", user.ID))
	return resp, nil
}

func (h *UserHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.Login")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", req.Email))

	h.logger.Debug("Login request received", zap.String("email", req.Email))

	user, token, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		h.logger.Error("Failed to login user", zap.Error(err), zap.String("email", req.Email))
		return nil, err
	}

	resp := &pb.LoginResponse{
		User:  modelToProto(user),
		Token: token,
	}

	h.logger.Info("Login completed successfully", zap.String("user_id", user.ID))
	return resp, nil
}

func (h *UserHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	ctx, span := h.tracer.Start(ctx, "UserHandler.ValidateToken")
	defer span.End()

	h.logger.Debug("ValidateToken request received")

	user, err := h.service.ValidateToken(ctx, req.Token)
	if err != nil {
		h.logger.Error("Failed to validate token", zap.Error(err))
		return nil, err
	}

	resp := &pb.ValidateTokenResponse{
		Valid: true,
		User:  modelToProto(user),
	}

	h.logger.Debug("ValidateToken completed successfully", zap.String("user_id", user.ID))
	return resp, nil
}

func modelToProto(user *model.User) *pb.User {
	if user == nil {
		return nil
	}

	return &pb.User{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}