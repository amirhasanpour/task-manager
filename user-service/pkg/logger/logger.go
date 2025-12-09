package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger *zap.Logger
	sugarLogger  *zap.SugaredLogger
)

type Config struct {
	Level           string
	Encoding        string
	OutputPaths     []string
	ErrorOutputPaths []string
}

func InitLogger(cfg Config) error {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          cfg.Encoding,
		EncoderConfig:     encoderConfig,
		OutputPaths:       cfg.OutputPaths,
		ErrorOutputPaths:  cfg.ErrorOutputPaths,
	}

	logger, err := config.Build()
	if err != nil {
		return err
	}

	globalLogger = logger
	sugarLogger = logger.Sugar()

	zap.ReplaceGlobals(logger)

	return nil
}

func GetLogger() *zap.Logger {
	if globalLogger == nil {
		panic("logger not initialized")
	}
	return globalLogger
}

func GetSugaredLogger() *zap.SugaredLogger {
	if sugarLogger == nil {
		panic("logger not initialized")
	}
	return sugarLogger
}

func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}