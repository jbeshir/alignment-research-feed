package app

import (
	"context"
	"fmt"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"os"
	"strconv"
	"strings"
	"time"
)

func MustGetEnvAsString(ctx context.Context, name string) string {
	s, exists := os.LookupEnv(name)
	if !exists {
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "environment variable missing", "variable_name", name)
		panic(fmt.Sprintf("missing environment variable [%s]", name))
	}

	return s
}

func MustGetEnvAsInt(ctx context.Context, name string) int {
	s := MustGetEnvAsString(ctx, name)

	v, err := strconv.Atoi(s)
	if err != nil {
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to parse environment variable as string",
			"variable_name", name,
			"variable_value", s,
		)
		panic(fmt.Sprintf("unable to parse environment variable as string [%s]: %s", name, s))
	}

	return v
}

func MustGetEnvAsBoolean(ctx context.Context, name string) bool {
	s := MustGetEnvAsString(ctx, name)

	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	default:
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to parse environment variable as boolean ('true'/'false')",
			"variable_name", name,
			"variable_value", s,
		)
		panic(fmt.Sprintf("unable to parse environment variable as boolean ('true'/'false') [%s]: %s", name, s))
	}
}

func MustGetEnvAsDuration(ctx context.Context, name string) time.Duration {
	s := MustGetEnvAsString(ctx, name)

	duration, err := time.ParseDuration(s)
	if err != nil {
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to parse environment variable as duration",
			"variable_name", name,
			"variable_value", s,
		)
		panic(fmt.Sprintf("unable to parse environment variable as duration [%s]: %s", name, s))
	}

	return duration
}
