package handler

import (
	"time"

	"github.com/amirhasanpour/task-manager/api-gateway/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// User models
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"max=200"`
}

type UpdateUserRequest struct {
	Username *string `json:"username,omitempty" binding:"omitempty,min=3,max=100"`
	Email    *string `json:"email,omitempty" binding:"omitempty,email"`
	Password *string `json:"password,omitempty" binding:"omitempty,min=8"`
	FullName *string `json:"full_name,omitempty" binding:"omitempty,max=200"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListUsersResponse struct {
	Users    []UserResponse `json:"users"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// Auth models
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"max=200"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

type ValidateTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

type ValidateTokenResponse struct {
	Valid bool         `json:"valid"`
	User  UserResponse `json:"user,omitempty"`
}

// Task models
type CreateTaskRequest struct {
	Title       string     `json:"title" binding:"required,min=1,max=255"`
	Description string     `json:"description"`
	Status      string     `json:"status" binding:"omitempty,oneof=TODO IN_PROGRESS DONE ARCHIVED"`
	Priority    string     `json:"priority" binding:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type UpdateTaskRequest struct {
	Title       *string    `json:"title,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty"`
	Status      *string    `json:"status,omitempty" binding:"omitempty,oneof=TODO IN_PROGRESS DONE ARCHIVED"`
	Priority    *string    `json:"priority,omitempty" binding:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type TaskResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ListTasksRequest struct {
	Page           int    `form:"page" binding:"omitempty,min=1"`
	PageSize       int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	FilterByStatus string `form:"filter_by_status" binding:"omitempty,oneof=TODO IN_PROGRESS DONE ARCHIVED"`
	FilterByPriority string `form:"filter_by_priority" binding:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	SortBy         string `form:"sort_by" binding:"omitempty,oneof=title status priority due_date created_at updated_at"`
	SortDesc       bool   `form:"sort_desc"`
}

type ListTasksResponse struct {
	Tasks    []TaskResponse `json:"tasks"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// Helper functions for conversion
func userProtoToResponse(user *proto.User) UserResponse {
	return UserResponse{
		ID:        user.Id,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		CreatedAt: user.CreatedAt.AsTime(),
		UpdatedAt: user.UpdatedAt.AsTime(),
	}
}

func taskProtoToResponse(task *proto.Task) TaskResponse {
	resp := TaskResponse{
		ID:          task.Id,
		UserID:      task.UserId,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status.String(),
		Priority:    task.Priority.String(),
		CreatedAt:   task.CreatedAt.AsTime(),
		UpdatedAt:   task.UpdatedAt.AsTime(),
	}

	if task.DueDate != nil {
		dueDate := task.DueDate.AsTime()
		resp.DueDate = &dueDate
	}

	return resp
}

func createTaskRequestToProto(req *CreateTaskRequest, userID string) *proto.CreateTaskRequest {
	protoReq := &proto.CreateTaskRequest{
		UserId:      userID,
		Title:       req.Title,
		Description: req.Description,
	}

	// Set status
	if req.Status != "" {
		protoReq.Status = proto.TaskStatus(proto.TaskStatus_value[req.Status])
	}

	// Set priority
	if req.Priority != "" {
		protoReq.Priority = proto.TaskPriority(proto.TaskPriority_value[req.Priority])
	}

	// Set due date
	if req.DueDate != nil {
		protoReq.DueDate = timestamppb.New(*req.DueDate)
	}

	return protoReq
}

func updateTaskRequestToProto(req *UpdateTaskRequest, taskID, userID string) *proto.UpdateTaskRequest {
	protoReq := &proto.UpdateTaskRequest{
		Id:     taskID,
		UserId: userID,
	}

	if req.Title != nil {
		protoReq.Title = *req.Title
	}
	if req.Description != nil {
		protoReq.Description = *req.Description
	}
	if req.Status != nil {
		protoReq.Status = proto.TaskStatus(proto.TaskStatus_value[*req.Status])
	}
	if req.Priority != nil {
		protoReq.Priority = proto.TaskPriority(proto.TaskPriority_value[*req.Priority])
	}
	if req.DueDate != nil {
		protoReq.DueDate = timestamppb.New(*req.DueDate)
	}

	return protoReq
}