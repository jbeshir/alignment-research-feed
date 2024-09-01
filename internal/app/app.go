package app

import (
	"context"
	"fmt"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mysql"
	"github.com/jbeshir/alignment-research-feed/internal/transport/web/router"
	"github.com/jbeshir/alignment-research-feed/internal/transport/web/server"
)

type Component interface {
	Run(ctx context.Context) error
}

func Setup(ctx context.Context) ([]Component, error) {
	dataset, err := setupDatasetRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("setting up dataset repository: %w", err)
	}

	httpRouter, err := router.MakeRouter(
		ctx,
		dataset,
		MustGetEnvAsString(ctx, "RSS_FEED_BASE_URL"),
		MustGetEnvAsString(ctx, "RSS_FEED_AUTHOR_NAME"),
		MustGetEnvAsString(ctx, "RSS_FEED_AUTHOR_EMAIL"),
		MustGetEnvAsDuration(ctx, "RSS_FEED_LATEST_CACHE_MAX_AGE"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create HTTP router: %w", err)
	}

	return []Component{
		&server.Server{
			TLSDisabled:       MustGetEnvAsBoolean(ctx, "HTTP_TLS_DISABLED"),
			TLSDisabledPort:   MustGetEnvAsInt(ctx, "PORT"),
			AutocertHostnames: MustGetEnvAsStrings(ctx, "HTTP_AUTOCERT_HOSTNAMES"),
			Router:            httpRouter,
		},
	}, nil
}

func setupDatasetRepository(ctx context.Context) (datasources.DatasetRepository, error) {
	db, err := mysql.Connect(ctx, MustGetEnvAsString(ctx, "MYSQL_URI"))
	if err != nil {
		return nil, fmt.Errorf("connecting to MySQL: %w", err)
	}

	return mysql.New(db), nil
}
