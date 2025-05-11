package domain

import (
	"context"
	"log/slog"
)

type contextKey string

const loggerContextKey contextKey = "logger"

func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	logger := ctx.Value(loggerContextKey)
	if logger == nil {
		logger = slog.Default()
	}

	return logger.(*slog.Logger)
}

const userContextKey contextKey = "user"

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userContextKey, userID)
}

func UserIDFromContext(ctx context.Context) string {
	userID := ctx.Value(userContextKey)
	if userID == nil {
		userID = ""
	}
	return userID.(string)
}
