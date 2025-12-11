package handler

import (
	"context"
	"time"

	"github.com/amirhasanpour/task-manager/todo-service/internal/model"
	"github.com/amirhasanpour/task-manager/todo-service/internal/repository"
	"github.com/amirhasanpour/task-manager/todo-service/internal/service"
	pb "github.com/amirhasanpour/task-manager/todo-service/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TaskHandler struct {
	pb.UnimplementedTodoServiceServer
	service service.TaskService
	logger  *zap.Logger
	tracer  trace.Tracer
}

func NewTaskHandler(service service.TaskService) *TaskHandler {
	return &TaskHandler{
		service: service,
		logger:  zap.L().Named("task_handler"),
		tracer:  trace.NewNoopTracerProvider().Tracer("noop"),
	}
}

func (h *TaskHandler) SetTracer(tracer trace.Tracer) {
	h.tracer = tracer
}

func (h *TaskHandler) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	ctx, span := h.tracer.Start(ctx, "TaskHandler.CreateTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.UserId),
		attribute.String("task.title", req.Title),
	)

	h.logger.Debug("CreateTask request received", 
		zap.String("user_id", req.UserId),
		zap.String("title", req.Title),
	)

	var dueDate *time.Time
	if req.DueDate != nil {
		dueDateValue := req.DueDate.AsTime()
		dueDate = &dueDateValue
	}

	serviceReq := &service.CreateTaskRequest{
		UserID:      req.UserId,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status.String(),
		Priority:    req.Priority.String(),
		DueDate:     dueDate,
	}

	task, err := h.service.CreateTask(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create task", zap.Error(err))
		return nil, err
	}

	resp := &pb.CreateTaskResponse{
		Task: modelToProto(task),
	}

	h.logger.Info("CreateTask completed successfully", 
		zap.String("task_id", task.ID),
		zap.String("user_id", req.UserId),
	)
	return resp, nil
}

func (h *TaskHandler) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	ctx, span := h.tracer.Start(ctx, "TaskHandler.GetTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", req.Id))

	h.logger.Debug("GetTask request received", zap.String("id", req.Id))

	task, err := h.service.GetTask(ctx, req.Id)
	if err != nil {
		h.logger.Error("Failed to get task", zap.Error(err), zap.String("id", req.Id))
		return nil, err
	}

	resp := &pb.GetTaskResponse{
		Task: modelToProto(task),
	}

	h.logger.Debug("GetTask completed successfully", zap.String("id", req.Id))
	return resp, nil
}

func (h *TaskHandler) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	ctx, span := h.tracer.Start(ctx, "TaskHandler.UpdateTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("task.id", req.Id),
		attribute.String("user.id", req.UserId),
	)

	h.logger.Debug("UpdateTask request received", 
		zap.String("id", req.Id),
		zap.String("user_id", req.UserId),
	)

	serviceReq := &service.UpdateTaskRequest{
		ID:     req.Id,
		UserID: req.UserId,
	}

	// Only set fields that are provided
	if req.Title != "" {
		serviceReq.Title = &req.Title
	}
	if req.Description != "" {
		serviceReq.Description = &req.Description
	}
	if req.Status != pb.TaskStatus(0) {
		statusStr := req.Status.String()
		serviceReq.Status = &statusStr
	}
	if req.Priority != pb.TaskPriority(0) {
		priorityStr := req.Priority.String()
		serviceReq.Priority = &priorityStr
	}
	if req.DueDate != nil {
		dueDate := req.DueDate.AsTime()
		serviceReq.DueDate = &dueDate
	}

	task, err := h.service.UpdateTask(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to update task", 
			zap.Error(err), 
			zap.String("id", req.Id),
		)
		return nil, err
	}

	resp := &pb.UpdateTaskResponse{
		Task: modelToProto(task),
	}

	h.logger.Info("UpdateTask completed successfully", zap.String("id", req.Id))
	return resp, nil
}

func (h *TaskHandler) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	ctx, span := h.tracer.Start(ctx, "TaskHandler.DeleteTask")
	defer span.End()

	span.SetAttributes(attribute.String("task.id", req.Id))

	h.logger.Debug("DeleteTask request received", zap.String("id", req.Id))

	err := h.service.DeleteTask(ctx, req.Id)
	if err != nil {
		h.logger.Error("Failed to delete task", zap.Error(err), zap.String("id", req.Id))
		return nil, err
	}

	resp := &pb.DeleteTaskResponse{
		Success: true,
	}

	h.logger.Info("DeleteTask completed successfully", zap.String("id", req.Id))
	return resp, nil
}

func (h *TaskHandler) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	ctx, span := h.tracer.Start(ctx, "TaskHandler.ListTasks")
	defer span.End()

	span.SetAttributes(
		attribute.Int("page", int(req.Page)),
		attribute.Int("page_size", int(req.PageSize)),
	)

	h.logger.Debug("ListTasks request received", 
		zap.Int32("page", req.Page),
		zap.Int32("page_size", req.PageSize),
		zap.String("filter_status", req.FilterByStatus),
		zap.String("filter_priority", req.FilterByPriority),
		zap.String("filter_user_id", req.FilterByUserId),
		zap.String("sort_by", req.SortBy),
		zap.Bool("sort_desc", req.SortDesc),
	)

	page := int(req.Page)
	pageSize := int(req.PageSize)

	filter := &repository.TaskFilter{
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
	}

	// Apply filters if provided
	if req.FilterByStatus != "" {
		filter.Status = &req.FilterByStatus
	}
	if req.FilterByPriority != "" {
		filter.Priority = &req.FilterByPriority
	}
	if req.FilterByUserId != "" {
		filter.UserID = &req.FilterByUserId
	}

	tasks, total, err := h.service.ListTasks(ctx, filter, page, pageSize)
	if err != nil {
		h.logger.Error("Failed to list tasks", zap.Error(err))
		return nil, err
	}

	protoTasks := make([]*pb.Task, len(tasks))
	for i, task := range tasks {
		protoTasks[i] = modelToProto(task)
	}

	resp := &pb.ListTasksResponse{
		Tasks:    protoTasks,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	h.logger.Debug("ListTasks completed successfully", 
		zap.Int("task_count", len(tasks)),
		zap.Int64("total", total),
	)
	return resp, nil
}

func (h *TaskHandler) ListTasksByUser(ctx context.Context, req *pb.ListTasksByUserRequest) (*pb.ListTasksByUserResponse, error) {
	ctx, span := h.tracer.Start(ctx, "TaskHandler.ListTasksByUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.UserId),
		attribute.Int("page", int(req.Page)),
		attribute.Int("page_size", int(req.PageSize)),
	)

	h.logger.Debug("ListTasksByUser request received", 
		zap.String("user_id", req.UserId),
		zap.Int32("page", req.Page),
		zap.Int32("page_size", req.PageSize),
		zap.String("filter_status", req.FilterByStatus),
		zap.String("filter_priority", req.FilterByPriority),
		zap.String("sort_by", req.SortBy),
		zap.Bool("sort_desc", req.SortDesc),
	)

	page := int(req.Page)
	pageSize := int(req.PageSize)

	filter := &repository.TaskFilter{
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
	}

	// Apply filters if provided
	if req.FilterByStatus != "" {
		filter.Status = &req.FilterByStatus
	}
	if req.FilterByPriority != "" {
		filter.Priority = &req.FilterByPriority
	}

	tasks, total, err := h.service.ListTasksByUser(ctx, req.UserId, filter, page, pageSize)
	if err != nil {
		h.logger.Error("Failed to list tasks by user", zap.Error(err))
		return nil, err
	}

	protoTasks := make([]*pb.Task, len(tasks))
	for i, task := range tasks {
		protoTasks[i] = modelToProto(task)
	}

	resp := &pb.ListTasksByUserResponse{
		Tasks:    protoTasks,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	h.logger.Debug("ListTasksByUser completed successfully", 
		zap.String("user_id", req.UserId),
		zap.Int("task_count", len(tasks)),
		zap.Int64("total", total),
	)
	return resp, nil
}

func modelToProto(task *model.Task) *pb.Task {
	if task == nil {
		return nil
	}

	protoTask := &pb.Task{
		Id:          task.ID,
		UserId:      task.UserID,
		Title:       task.Title,
		Description: task.Description,
		Status:      protoStatus(task.ToProtoStatus()),
		Priority:    protoPriority(task.ToProtoPriority()),
		CreatedAt:   timestamppb.New(task.CreatedAt),
		UpdatedAt:   timestamppb.New(task.UpdatedAt),
	}

	if task.DueDate != nil {
		protoTask.DueDate = timestamppb.New(*task.DueDate)
	}

	return protoTask
}

func protoStatus(status string) pb.TaskStatus {
	switch status {
	case "TODO":
		return pb.TaskStatus_TODO
	case "IN_PROGRESS":
		return pb.TaskStatus_IN_PROGRESS
	case "DONE":
		return pb.TaskStatus_DONE
	case "ARCHIVED":
		return pb.TaskStatus_ARCHIVED
	default:
		return pb.TaskStatus_TODO
	}
}

func protoPriority(priority string) pb.TaskPriority {
	switch priority {
	case "LOW":
		return pb.TaskPriority_LOW
	case "MEDIUM":
		return pb.TaskPriority_MEDIUM
	case "HIGH":
		return pb.TaskPriority_HIGH
	case "URGENT":
		return pb.TaskPriority_URGENT
	default:
		return pb.TaskPriority_MEDIUM
	}
}