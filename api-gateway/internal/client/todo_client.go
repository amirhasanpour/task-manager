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

type TodoClient interface {
	CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error)
	GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error)
	UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error)
	DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error)
	ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error)
	ListTasksByUser(ctx context.Context, req *pb.ListTasksByUserRequest) (*pb.ListTasksByUserResponse, error)
	Close() error
}

type todoClient struct {
	conn   *grpc.ClientConn
	client pb.TodoServiceClient
	logger *zap.Logger
	tracer trace.Tracer
}

type TodoConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
}

func NewTodoClient(cfg TodoConfig) (TodoClient, error) {
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(cfg.Timeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to todo service: %w", err)
	}

	client := pb.NewTodoServiceClient(conn)
	
	logger := zap.L().Named("todo_client")
	logger.Info("Connected to todo service", zap.String("address", address))

	return &todoClient{
		conn:   conn,
		client: client,
		logger: logger,
		tracer: otel.Tracer("todo-client"),
	}, nil
}

func (c *todoClient) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	ctx, span := c.tracer.Start(ctx, "TodoClient.CreateTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.UserId),
		attribute.String("task.title", req.Title),
	)
	c.logger.Debug("Creating task", zap.String("user_id", req.UserId), zap.String("title", req.Title))
	return c.client.CreateTask(ctx, req)
}

func (c *todoClient) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	ctx, span := c.tracer.Start(ctx, "TodoClient.GetTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", req.Id))
	c.logger.Debug("Getting task", zap.String("id", req.Id))
	return c.client.GetTask(ctx, req)
}

func (c *todoClient) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	ctx, span := c.tracer.Start(ctx, "TodoClient.UpdateTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("task.id", req.Id),
		attribute.String("user.id", req.UserId),
	)
	c.logger.Debug("Updating task", zap.String("id", req.Id), zap.String("user_id", req.UserId))
	return c.client.UpdateTask(ctx, req)
}

func (c *todoClient) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	ctx, span := c.tracer.Start(ctx, "TodoClient.DeleteTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", req.Id))
	c.logger.Debug("Deleting task", zap.String("id", req.Id))
	return c.client.DeleteTask(ctx, req)
}

func (c *todoClient) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	ctx, span := c.tracer.Start(ctx, "TodoClient.ListTasks")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", int(req.Page)),
		attribute.Int("page_size", int(req.PageSize)),
	)
	c.logger.Debug("Listing tasks", 
		zap.Int32("page", req.Page),
		zap.Int32("page_size", req.PageSize),
		zap.String("filter_status", req.FilterByStatus),
		zap.String("filter_priority", req.FilterByPriority),
	)
	return c.client.ListTasks(ctx, req)
}

func (c *todoClient) ListTasksByUser(ctx context.Context, req *pb.ListTasksByUserRequest) (*pb.ListTasksByUserResponse, error) {
	ctx, span := c.tracer.Start(ctx, "TodoClient.ListTasksByUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.UserId),
		attribute.Int("page", int(req.Page)),
		attribute.Int("page_size", int(req.PageSize)),
	)
	c.logger.Debug("Listing tasks by user", 
		zap.String("user_id", req.UserId),
		zap.Int32("page", req.Page),
		zap.Int32("page_size", req.PageSize),
	)
	return c.client.ListTasksByUser(ctx, req)
}

func (c *todoClient) Close() error {
	c.logger.Info("Closing todo client connection")
	return c.conn.Close()
}