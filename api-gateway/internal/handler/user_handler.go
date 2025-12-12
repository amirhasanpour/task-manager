package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/client"
	pb "github.com/amirhasanpour/task-manager/api-gateway/proto"
	"go.uber.org/zap"
)

type UserHandler struct {
	userClient client.UserClient
	logger     *zap.Logger
}

func NewUserHandler(userClient client.UserClient) *UserHandler {
	return &UserHandler{
		userClient: userClient,
		logger:     zap.L().Named("user_handler"),
	}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid create user request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to proto request
	protoReq := &pb.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}

	// Call user service
	resp, err := h.userClient.CreateUser(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		// TODO: Handle specific gRPC errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Convert response
	userResp := userProtoToResponse(resp.User)

	h.logger.Info("User created successfully", zap.String("user_id", resp.User.Id))
	c.JSON(http.StatusCreated, userResp)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Convert to proto request
	protoReq := &pb.GetUserRequest{
		Id: userID,
	}

	// Call user service
	resp, err := h.userClient.GetUser(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err), zap.String("user_id", userID))
		// TODO: Handle not found errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Convert response
	userResp := userProtoToResponse(resp.User)

	h.logger.Debug("User retrieved", zap.String("user_id", userID))
	c.JSON(http.StatusOK, userResp)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid update user request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to proto request
	protoReq := &pb.UpdateUserRequest{
		Id:       userID,
		Username: "",
		Email:    "",
		FullName: "",
		Password: "",
	}

	// Only set fields that are provided
	if req.Username != nil {
		protoReq.Username = *req.Username
	}
	if req.Email != nil {
		protoReq.Email = *req.Email
	}
	if req.FullName != nil {
		protoReq.FullName = *req.FullName
	}
	if req.Password != nil {
		protoReq.Password = *req.Password
	}

	// Call user service
	resp, err := h.userClient.UpdateUser(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to update user", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Convert response
	userResp := userProtoToResponse(resp.User)

	h.logger.Info("User updated successfully", zap.String("user_id", userID))
	c.JSON(http.StatusOK, userResp)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Convert to proto request
	protoReq := &pb.DeleteUserRequest{
		Id: userID,
	}

	// Call user service
	resp, err := h.userClient.DeleteUser(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to delete user", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	h.logger.Info("User deleted successfully", zap.String("user_id", userID))
	c.JSON(http.StatusOK, gin.H{"success": resp.Success})
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

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

	// Convert to proto request
	protoReq := &pb.ListUsersRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	}

	// Call user service
	resp, err := h.userClient.ListUsers(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	// Convert response
	users := make([]UserResponse, len(resp.Users))
	for i, user := range resp.Users {
		users[i] = userProtoToResponse(user)
	}

	listResp := ListUsersResponse{
		Users:    users,
		Total:    int64(resp.Total),
		Page:     int(resp.Page),
		PageSize: int(resp.PageSize),
	}

	h.logger.Debug("Users listed", zap.Int("count", len(users)))
	c.JSON(http.StatusOK, listResp)
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Convert to proto request
	protoReq := &pb.GetUserRequest{
		Id: userID.(string),
	}

	// Call user service
	resp, err := h.userClient.GetUser(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to get current user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Convert response
	userResp := userProtoToResponse(resp.User)

	h.logger.Debug("Current user retrieved", zap.String("user_id", userID.(string)))
	c.JSON(http.StatusOK, userResp)
}

func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid update current user request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to proto request
	protoReq := &pb.UpdateUserRequest{
		Id:       userID.(string),
		Username: "",
		Email:    "",
		FullName: "",
		Password: "",
	}

	// Only set fields that are provided
	if req.Username != nil {
		protoReq.Username = *req.Username
	}
	if req.Email != nil {
		protoReq.Email = *req.Email
	}
	if req.FullName != nil {
		protoReq.FullName = *req.FullName
	}
	if req.Password != nil {
		protoReq.Password = *req.Password
	}

	// Call user service
	resp, err := h.userClient.UpdateUser(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to update current user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Convert response
	userResp := userProtoToResponse(resp.User)

	h.logger.Info("Current user updated successfully", zap.String("user_id", userID.(string)))
	c.JSON(http.StatusOK, userResp)
}