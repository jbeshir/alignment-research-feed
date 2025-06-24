package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mysql"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/pinecone"
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

	similarity, err := setupSimilarityRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("setting up similarity repository: %w", err)
	}

	authMiddleware, err := setupAuthMiddleware(ctx)
	if err != nil {
		return nil, fmt.Errorf("setting up auth middleware: %w", err)
	}

	httpRouter, err := router.MakeRouter(
		dataset,
		similarity,
		MustGetEnvAsString(ctx, "RSS_FEED_BASE_URL"),
		MustGetEnvAsString(ctx, "RSS_FEED_AUTHOR_NAME"),
		MustGetEnvAsString(ctx, "RSS_FEED_AUTHOR_EMAIL"),
		MustGetEnvAsDuration(ctx, "RSS_FEED_LATEST_CACHE_MAX_AGE"),
		authMiddleware,
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

func setupSimilarityRepository(ctx context.Context) (datasources.SimilarArticleLister, error) {
	switch driver := MustGetEnvAsString(ctx, "SIMILARITY_DRIVER"); driver {
	case "null":
		return datasources.NullSimilarArticleLister{}, nil
	case "pinecone":
		similarity, err := pinecone.NewClient(ctx, MustGetEnvAsString(ctx, "PINECONE_API_KEY"))
		if err != nil {
			return nil, fmt.Errorf("connecting to pinecone: %w", err)
		}

		return similarity, nil
	default:
		return nil, fmt.Errorf("unknown similarity driver [%s]", driver)
	}
}

func setupAuthMiddleware(ctx context.Context) (func(http.Handler) http.Handler, error) {
	switch driver := MustGetEnvAsString(ctx, "AUTH_DRIVER"); driver {
	case "null":
		return func(next http.Handler) http.Handler {
			return next
		}, nil
	case "auth0":
		return router.SetupAuth0Middleware(
			MustGetEnvAsString(ctx, "AUTH0_DOMAIN"),
			MustGetEnvAsString(ctx, "AUTH0_AUDIENCE"))
	default:
		return nil, fmt.Errorf("unknown auth driver [%s]", driver)
	}
}
