package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/client"
	pb "github.com/amirhasanpour/task-manager/api-gateway/proto"
	"go.uber.org/zap"
)

type TaskHandler struct {
	todoClient client.TodoClient
	logger     *zap.Logger
}

func NewTaskHandler(todoClient client.TodoClient) *TaskHandler {
	return &TaskHandler{
		todoClient: todoClient,
		logger:     zap.L().Named("task_handler"),
	}
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid create task request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to proto request
	protoReq := createTaskRequestToProto(&req, userID.(string))

	// Call todo service
	resp, err := h.todoClient.CreateTask(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to create task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Convert response
	taskResp := taskProtoToResponse(resp.Task)

	h.logger.Info("Task created successfully", 
		zap.String("task_id", resp.Task.Id),
		zap.String("user_id", userID.(string)),
	)
	c.JSON(http.StatusCreated, taskResp)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	// Convert to proto request
	protoReq := &pb.GetTaskRequest{
		Id: taskID,
	}

	// Call todo service
	resp, err := h.todoClient.GetTask(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to get task", zap.Error(err), zap.String("task_id", taskID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		return
	}

	// Convert response
	taskResp := taskProtoToResponse(resp.Task)

	h.logger.Debug("Task retrieved", zap.String("task_id", taskID))
	c.JSON(http.StatusOK, taskResp)
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid update task request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to proto request
	protoReq := updateTaskRequestToProto(&req, taskID, userID.(string))

	// Call todo service
	resp, err := h.todoClient.UpdateTask(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to update task", zap.Error(err), zap.String("task_id", taskID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Convert response
	taskResp := taskProtoToResponse(resp.Task)

	h.logger.Info("Task updated successfully", zap.String("task_id", taskID))
	c.JSON(http.StatusOK, taskResp)
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	// Convert to proto request
	protoReq := &pb.DeleteTaskRequest{
		Id: taskID,
	}

	// Call todo service
	resp, err := h.todoClient.DeleteTask(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to delete task", zap.Error(err), zap.String("task_id", taskID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	h.logger.Info("Task deleted successfully", zap.String("task_id", taskID))
	c.JSON(http.StatusOK, gin.H{"success": resp.Success})
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	// Parse query parameters
	var query ListTasksRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Debug("Invalid list tasks query", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 10
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	// Convert to proto request
	protoReq := &pb.ListTasksRequest{
		Page:              int32(query.Page),
		PageSize:          int32(query.PageSize),
		FilterByStatus:    query.FilterByStatus,
		FilterByPriority:  query.FilterByPriority,
		FilterByUserId:    "", // Admin only - will be empty for regular users
		SortBy:            query.SortBy,
		SortDesc:          query.SortDesc,
	}

	// Call todo service
	resp, err := h.todoClient.ListTasks(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to list tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		return
	}

	// Convert response
	tasks := make([]TaskResponse, len(resp.Tasks))
	for i, task := range resp.Tasks {
		tasks[i] = taskProtoToResponse(task)
	}

	listResp := ListTasksResponse{
		Tasks:    tasks,
		Total:    int64(resp.Total),
		Page:     int(resp.Page),
		PageSize: int(resp.PageSize),
	}

	h.logger.Debug("Tasks listed", zap.Int("count", len(tasks)))
	c.JSON(http.StatusOK, listResp)
}

func (h *TaskHandler) ListMyTasks(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Parse query parameters
	var query ListTasksRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Debug("Invalid list my tasks query", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 10
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	// Convert to proto request
	protoReq := &pb.ListTasksByUserRequest{
		UserId:           userID.(string),
		Page:             int32(query.Page),
		PageSize:         int32(query.PageSize),
		FilterByStatus:   query.FilterByStatus,
		FilterByPriority: query.FilterByPriority,
		SortBy:           query.SortBy,
		SortDesc:         query.SortDesc,
	}

	// Call todo service
	resp, err := h.todoClient.ListTasksByUser(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to list my tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		return
	}

	// Convert response
	tasks := make([]TaskResponse, len(resp.Tasks))
	for i, task := range resp.Tasks {
		tasks[i] = taskProtoToResponse(task)
	}

	listResp := ListTasksResponse{
		Tasks:    tasks,
		Total:    int64(resp.Total),
		Page:     int(resp.Page),
		PageSize: int(resp.PageSize),
	}

	h.logger.Debug("My tasks listed", 
		zap.String("user_id", userID.(string)),
		zap.Int("count", len(tasks)),
	)
	c.JSON(http.StatusOK, listResp)
}