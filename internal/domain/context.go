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

// AuthMethod represents the authentication method used for a request.
type AuthMethod string

const (
	AuthMethodNone     AuthMethod = ""
	AuthMethodAuth0    AuthMethod = "auth0"
	AuthMethodAPIToken AuthMethod = "api_token"
)

const authMethodContextKey contextKey = "auth_method"

func ContextWithAuthMethod(ctx context.Context, method AuthMethod) context.Context {
	return context.WithValue(ctx, authMethodContextKey, method)
}

func AuthMethodFromContext(ctx context.Context) AuthMethod {
	method := ctx.Value(authMethodContextKey)
	if method == nil {
		return AuthMethodNone
	}
	return method.(AuthMethod)
}
