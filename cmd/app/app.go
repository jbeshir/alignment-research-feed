package main

import (
	"context"
	"fmt"
	"github.com/jbeshir/alignment-research-feed/internal/app"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"os"
)

import _ "github.com/joho/godotenv/autoload"

func main() {
	ctx := context.Background()

	var logLevel slog.Level
	logLevelStr := app.MustGetEnvAsString(ctx, "LOG_LEVEL")
	if err := logLevel.UnmarshalText([]byte(logLevelStr)); err != nil {
		panic(fmt.Sprintf("unable to setup logger, LOG_LEVEL not recognised [%s]", logLevelStr))
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)
	ctx = domain.ContextWithLogger(ctx, logger)

	components, err := app.Setup(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "unable to setup components", "error", err)
		os.Exit(1)
	}

	grp, grpCtx := errgroup.WithContext(ctx)
	for _, c := range components {
		grp.Go(func() error {
			return c.Run(grpCtx)
		})
	}

	if err = grp.Wait(); err != nil {
		logger.ErrorContext(ctx, "shutting down due to error", "error", err)
		os.Exit(1)
	}
}
