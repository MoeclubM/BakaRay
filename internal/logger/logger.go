package logger

import (
	"log/slog"
	"os"
)

var (
	Log *slog.Logger
)

func Init(level string) error {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	Log = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(Log)

	return nil
}

func WithRequestID(requestID string) *slog.Logger {
	return Log.With("request_id", requestID)
}

func WithUserID(userID uint) *slog.Logger {
	return Log.With("user_id", userID)
}

func WithComponent(component string) *slog.Logger {
	return Log.With("component", component)
}

func WithContext(requestID string, userID uint, component string) *slog.Logger {
	return Log.With("request_id", requestID, "user_id", userID, "component", component)
}

func WithAdminContext(requestID string, userID uint, component string) *slog.Logger {
	return WithContext(requestID, userID, component)
}

func Error(msg string, err error, args ...any) {
	args = append(args, "error", err)
	Log.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}
