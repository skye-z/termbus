package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	globalLogger *zap.Logger
	sugarLogger  *zap.SugaredLogger
	auditLogger  *zap.Logger
	once         sync.Once
)

// Init 初始化日志系统
func Init(config *LogConfig) error {
	var initErr error
	once.Do(func() {
		globalLogger, initErr = createLogger(config)
		if initErr != nil {
			return
		}
		sugarLogger = globalLogger.Sugar()
		auditLogger, initErr = createAuditLogger(config)
	})
	return initErr
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string
	OutputPath string
	MaxSize    int
	MaxBackups int
	MaxAge     int
}

// createLogger 创建应用日志
func createLogger(config *LogConfig) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if config.OutputPath != "" {
		if err := os.MkdirAll(filepath.Dir(config.OutputPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		writer := &lumberjack.Logger{
			Filename:   config.OutputPath,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   true,
		}
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(writer),
			level,
		)
		return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)
	return zap.New(core, zap.AddCaller()), nil
}

// createAuditLogger 创建审计日志
func createAuditLogger(config *LogConfig) (*zap.Logger, error) {
	auditPath := filepath.Join(filepath.Dir(config.OutputPath), "audit.log")
	if err := os.MkdirAll(filepath.Dir(auditPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	writer := &lumberjack.Logger{
		Filename:   auditPath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge * 6,
		Compress:   true,
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:     "time",
		LevelKey:    "level",
		MessageKey:  "msg",
		LineEnding:  zapcore.DefaultLineEnding,
		EncodeLevel: zapcore.CapitalLevelEncoder,
		EncodeTime:  zapcore.ISO8601TimeEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(writer),
		zapcore.InfoLevel,
	)

	return zap.New(core), nil
}

// GetLogger 获取logger
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		globalLogger, _ = zap.NewDevelopment()
	}
	return globalLogger
}

// GetSugar 获取sugar logger
func GetSugar() *zap.SugaredLogger {
	if sugarLogger == nil {
		sugarLogger = GetLogger().Sugar()
	}
	return sugarLogger
}

// GetAuditLogger 获取审计logger
func GetAuditLogger() *zap.Logger {
	if auditLogger == nil {
		auditLogger, _ = createAuditLogger(&LogConfig{
			OutputPath: "audit.log",
			MaxSize:    100,
			MaxBackups: 30,
			MaxAge:     90,
		})
	}
	return auditLogger
}

// AuditLog 审计日志记录
func AuditLog(event, user, host, details string) {
	GetAuditLogger().Info("audit",
		zap.String("event", event),
		zap.String("user", user),
		zap.String("host", host),
		zap.String("details", details),
	)
}

// AuditLogWithFields 带字段的审计日志
func AuditLogWithFields(fields map[string]interface{}) {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	GetAuditLogger().Info("audit", zapFields...)
}
