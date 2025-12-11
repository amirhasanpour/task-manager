package interceptor

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoggingInterceptor struct {
	logger *zap.Logger
}

func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{
		logger: zap.L().Named("grpc_interceptor"),
	}
}

func (li *LoggingInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startTime := time.Now()
		
		// Extract trace ID
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()
		
		// Log request
		li.logger.Debug("GRPC request started",
			zap.String("method", info.FullMethod),
			zap.String("trace_id", traceID),
		)
		
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
		
		// Prepare log fields
		fields := []zapcore.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("status", statusCode.String()),
			zap.String("trace_id", traceID),
		}
		
		// Log based on status code
		switch  statusCode{
		case codes.OK:
			li.logger.Info("GRPC request completed", fields...)
		case codes.Internal:
			li.logger.Error("GRPC request failed with internal error", append(fields, zap.Error(err))...)
		case codes.Unauthenticated, codes.PermissionDenied:
			li.logger.Warn("GRPC request failed with auth error", append(fields, zap.Error(err))...)
		default:
			li.logger.Warn("GRPC request failed", append(fields, zap.Error(err))...)
		}
		
		return resp, err
	}
}