package client

import (
	"context"
	"fmt"
	"time"

	pb "github.com/amirhasanpour/task-manager/api-gateway/proto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UserClient interface defines the methods for user service client
type UserClient interface {
	CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error)
	GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error)
	UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error)
	DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error)
	ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error)
	Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error)
	Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error)
	ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error)
	Close() error
}

// userClientImpl is the actual implementation
type userClientImpl struct {
	conn   *grpc.ClientConn
	client pb.UserServiceClient
	logger *zap.Logger
	tracer trace.Tracer
}

// UserClientImpl is the exported type
type UserClientImpl struct {
	*userClientImpl
}

type UserConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
}

func NewUserClient(cfg UserConfig) (UserClient, error) {
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(cfg.Timeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	client := pb.NewUserServiceClient(conn)
	
	logger := zap.L().Named("user_client")
	logger.Info("Connected to user service", zap.String("address", address))

	impl := &userClientImpl{
		conn:   conn,
		client: client,
		logger: logger,
		tracer: otel.Tracer("user-client"),
	}

	return &UserClientImpl{impl}, nil
}

func (c *userClientImpl) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	ctx, span := c.tracer.Start(ctx, "UserClient.CreateUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("user.username", req.Username),
	)

	c.logger.Debug("Creating user", zap.String("email", req.Email))
	return c.client.CreateUser(ctx, req)
}

func (c *userClientImpl) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	ctx, span := c.tracer.Start(ctx, "UserClient.GetUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.Id))
	c.logger.Debug("Getting user", zap.String("id", req.Id))
	return c.client.GetUser(ctx, req)
}

func (c *userClientImpl) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	ctx, span := c.tracer.Start(ctx, "UserClient.UpdateUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.Id))
	c.logger.Debug("Updating user", zap.String("id", req.Id))
	return c.client.UpdateUser(ctx, req)
}

func (c *userClientImpl) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	ctx, span := c.tracer.Start(ctx, "UserClient.DeleteUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.Id))
	c.logger.Debug("Deleting user", zap.String("id", req.Id))
	return c.client.DeleteUser(ctx, req)
}

func (c *userClientImpl) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	ctx, span := c.tracer.Start(ctx, "UserClient.ListUsers")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", int(req.Page)),
		attribute.Int("page_size", int(req.PageSize)),
	)
	c.logger.Debug("Listing users", zap.Int32("page", req.Page), zap.Int32("page_size", req.PageSize))
	return c.client.ListUsers(ctx, req)
}

func (c *userClientImpl) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	ctx, span := c.tracer.Start(ctx, "UserClient.Register")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("user.username", req.Username),
	)
	c.logger.Debug("Registering user", zap.String("email", req.Email))
	return c.client.Register(ctx, req)
}

func (c *userClientImpl) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	ctx, span := c.tracer.Start(ctx, "UserClient.Login")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", req.Email))
	c.logger.Debug("User login", zap.String("email", req.Email))
	return c.client.Login(ctx, req)
}

func (c *userClientImpl) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	ctx, span := c.tracer.Start(ctx, "UserClient.ValidateToken")
	defer span.End()

	c.logger.Debug("Validating token")
	return c.client.ValidateToken(ctx, req)
}

func (c *userClientImpl) Close() error {
	c.logger.Info("Closing user client connection")
	return c.conn.Close()
}