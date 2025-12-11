package interceptor

import (
	"context"
	"time"

	"github.com/amirhasanpour/task-manager/todo-service/pkg/metrics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MetricsInterceptor struct {
	metrics *metrics.Metrics
	logger  *zap.Logger
}

func NewMetricsInterceptor(m *metrics.Metrics) *MetricsInterceptor {
	return &MetricsInterceptor{
		metrics: m,
		logger:  zap.L().Named("metrics_interceptor"),
	}
}

func (mi *MetricsInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startTime := time.Now()
		
		// Extract method name from full method
		method := extractMethodName(info.FullMethod)
		service := "todo-service"
		
		// Call handler
		resp, err := handler(ctx, req)
		
		// Calculate duration
		duration := time.Since(startTime)
		
		// Get status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Unknown
			}
		}
		
		// Record metrics
		mi.metrics.RecordRequest(service, "unary", method, int(statusCode), duration)
		
		// Record specific errors
		if err != nil {
			switch statusCode {
			case codes.Internal:
				mi.metrics.IncrementDatabaseErrors()
			case codes.Unauthenticated, codes.PermissionDenied:
				// Not incrementing auth errors as todo-service doesn't handle auth directly
			case codes.InvalidArgument, codes.AlreadyExists, codes.NotFound:
				mi.metrics.IncrementValidationErrors()
			}
		}
		
		mi.logger.Debug("Request processed",
			zap.String("method", method),
			zap.Duration("duration", duration),
			zap.String("status", statusCode.String()),
		)
		
		return resp, err
	}
}

func extractMethodName(fullMethod string) string {
	// fullMethod format: /package.Service/Method
	// Extract just the Method name
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return fullMethod
}