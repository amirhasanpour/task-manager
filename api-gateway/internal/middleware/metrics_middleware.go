package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/amirhasanpour/task-manager/api-gateway/pkg/metrics"
)

type MetricsMiddleware struct {
	metrics *metrics.Metrics
}

func NewMetricsMiddleware(m *metrics.Metrics) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: m,
	}
}

func (m *MetricsMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Increment active connections
		m.metrics.IncrementActiveConnections()
		defer m.metrics.DecrementActiveConnections()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)
		
		// Get status code
		statusCode := c.Writer.Status()
		
		// Extract endpoint (remove IDs for metrics)
		endpoint := m.extractEndpoint(c.Request.URL.Path)
		
		// Record metrics
		m.metrics.RecordRequest("api-gateway", c.Request.Method, endpoint, statusCode, duration)
	}
}

func (m *MetricsMiddleware) extractEndpoint(path string) string {
	// Simplify path for metrics (remove IDs)
	// Example: /api/v1/users/123 -> /api/v1/users/:id
	// This groups similar endpoints together in metrics
	
	// Simple implementation - for production, you might want a more sophisticated approach
	// For now, we'll use the full path but this should be enhanced based on your route patterns
	return path
}