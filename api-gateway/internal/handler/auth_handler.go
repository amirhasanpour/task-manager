package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/amirhasanpour/task-manager/api-gateway/internal/client"
	pb "github.com/amirhasanpour/task-manager/api-gateway/proto"
	"go.uber.org/zap"
)

type AuthHandler struct {
	userClient client.UserClient
	logger     *zap.Logger
}

func NewAuthHandler(userClient client.UserClient) *AuthHandler {
	return &AuthHandler{
		userClient: userClient,
		logger:     zap.L().Named("auth_handler"),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid register request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to proto request
	protoReq := &pb.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}

	// Call user service
	resp, err := h.userClient.Register(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to register user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	// Convert response
	authResp := AuthResponse{
		User:  userProtoToResponse(resp.User),
		Token: resp.Token,
	}

	h.logger.Info("User registered successfully", zap.String("user_id", resp.User.Id))
	c.JSON(http.StatusCreated, authResp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to proto request
	protoReq := &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	// Call user service
	resp, err := h.userClient.Login(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to login user", zap.Error(err), zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Convert response
	authResp := AuthResponse{
		User:  userProtoToResponse(resp.User),
		Token: resp.Token,
	}

	h.logger.Info("User logged in successfully", zap.String("user_id", resp.User.Id))
	c.JSON(http.StatusOK, authResp)
}

func (h *AuthHandler) ValidateToken(c *gin.Context) {
	var req ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("Invalid validate token request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to proto request
	protoReq := &pb.ValidateTokenRequest{
		Token: req.Token,
	}

	// Call user service
	resp, err := h.userClient.ValidateToken(c.Request.Context(), protoReq)
	if err != nil {
		h.logger.Error("Failed to validate token", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Convert response
	validateResp := ValidateTokenResponse{
		Valid: resp.Valid,
	}

	if resp.Valid && resp.User != nil {
		validateResp.User = userProtoToResponse(resp.User)
	}

	h.logger.Debug("Token validated", zap.Bool("valid", resp.Valid))
	c.JSON(http.StatusOK, validateResp)
}