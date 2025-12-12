package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type HealthHandler struct {
	logger *zap.Logger
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		logger: zap.L().Named("health_handler"),
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	h.logger.Debug("Health check requested")
	
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"service": "api-gateway",
		"timestamp": c.GetTime("request_time"),
	})
}