package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggingMiddleware struct {
	logger *zap.Logger
}

func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: zap.L().Named("http_logger"),
	}
}

func (m *LoggingMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Extract trace ID
		span := trace.SpanFromContext(c.Request.Context())
		traceID := span.SpanContext().TraceID().String()
		
		// Log request start
		m.logger.Debug("HTTP request started",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("trace_id", traceID),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)
		
		// Get status code
		statusCode := c.Writer.Status()
		
		// Prepare log fields
		fields := []zapcore.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("trace_id", traceID),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Add request ID if available
		if requestID := c.GetString("X-Request-ID"); requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}

		// Add user ID if available
		if userID := c.GetString("user_id"); userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}

		// Log based on status code
		switch {
		case statusCode >= 500:
			m.logger.Error("HTTP request failed with server error", fields...)
		case statusCode >= 400:
			m.logger.Warn("HTTP request failed with client error", fields...)
		default:
			m.logger.Info("HTTP request completed", fields...)
		}
	}
}