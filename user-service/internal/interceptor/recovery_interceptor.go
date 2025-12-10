package interceptor

import (
	"context"
	"runtime/debug"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RecoveryInterceptor struct {
	logger *zap.Logger
}

func NewRecoveryInterceptor() *RecoveryInterceptor {
	return &RecoveryInterceptor{
		logger: zap.L().Named("recovery_interceptor"),
	}
}
func (ri *RecoveryInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				ri.logger.Error("GRPC server panic recovered",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
					zap.String("method", info.FullMethod),
				)
				
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		
		return handler(ctx, req)
	}
}