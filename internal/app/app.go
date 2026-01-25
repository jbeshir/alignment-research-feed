package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/command"
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

	authMiddleware, err := setupAuthMiddleware(ctx, dataset)
	if err != nil {
		return nil, fmt.Errorf("setting up auth middleware: %w", err)
	}

	createAPITokenCmd := command.NewCreateAPIToken(dataset, dataset)

	generateRecommendationsCmd := command.NewGenerateRecommendations(
		similarity,
		dataset,
		dataset,
		dataset,
		DefaultGenerateRecommendationsConfig(),
	)

	recommendArticlesCmd := command.NewRecommendArticles(
		generateRecommendationsCmd,
		dataset,
		dataset,
		dataset,
		dataset,
		dataset,
		DefaultRecommendArticlesConfig(),
	)

	httpRouter, err := router.MakeRouter(
		dataset,
		similarity,
		MustGetEnvAsString(ctx, "RSS_FEED_BASE_URL"),
		MustGetEnvAsString(ctx, "RSS_FEED_AUTHOR_NAME"),
		MustGetEnvAsString(ctx, "RSS_FEED_AUTHOR_EMAIL"),
		MustGetEnvAsDuration(ctx, "RSS_FEED_LATEST_CACHE_MAX_AGE"),
		authMiddleware,
		createAPITokenCmd,
		recommendArticlesCmd,
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

func setupSimilarityRepository(ctx context.Context) (datasources.SimilarityRepository, error) {
	switch driver := MustGetEnvAsString(ctx, "SIMILARITY_DRIVER"); driver {
	case "null":
		return datasources.NullSimilarityRepository{}, nil
	case "pinecone":
		client, err := pinecone.NewClient(
			ctx,
			MustGetEnvAsString(ctx, "PINECONE_API_KEY"),
			MustGetEnvAsString(ctx, "PINECONE_INDEX_NAME"),
		)
		if err != nil {
			return nil, fmt.Errorf("connecting to pinecone: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unknown similarity driver [%s]", driver)
	}
}

func setupAuthMiddleware(
	ctx context.Context, dataset datasources.DatasetRepository,
) (func(http.Handler) http.Handler, error) {
	var validators []router.AuthValidator

	for _, driver := range MustGetEnvAsStrings(ctx, "AUTH_DRIVERS") {
		switch driver {
		case "":
			// Skip empty strings (e.g., from splitting an empty AUTH_DRIVERS)
		case "auth0":
			v, err := router.NewAuth0Validator(
				MustGetEnvAsString(ctx, "AUTH0_DOMAIN"),
				MustGetEnvAsString(ctx, "AUTH0_AUDIENCE"),
			)
			if err != nil {
				return nil, fmt.Errorf("creating Auth0 validator: %w", err)
			}
			validators = append(validators, v)
		case "api_token":
			validators = append(validators, router.NewAPITokenValidator(ctx, dataset, dataset))
		default:
			return nil, fmt.Errorf("unknown auth driver [%s]", driver)
		}
	}

	return router.NewAuthMiddleware(validators), nil
}
