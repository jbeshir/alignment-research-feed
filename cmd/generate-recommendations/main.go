package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"

	"github.com/jbeshir/alignment-research-feed/internal/app"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mysql"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/pinecone"
	"github.com/jbeshir/alignment-research-feed/internal/domain"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx := context.Background()

	// Setup logger
	logLevel := slog.LevelInfo
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		if err := logLevel.UnmarshalText([]byte(lvl)); err != nil {
			fmt.Fprintf(os.Stderr, "invalid LOG_LEVEL: %s\n", lvl)
			os.Exit(1)
		}
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)
	ctx = domain.ContextWithLogger(ctx, logger)

	if err := run(ctx); err != nil {
		logger.ErrorContext(ctx, "recommendation generation failed", "error", err)
		os.Exit(1)
	}

	logger.InfoContext(ctx, "recommendation generation completed successfully")
}

func run(ctx context.Context) error {
	// Connect to MySQL
	mysqlURI := os.Getenv("MYSQL_URI")
	if mysqlURI == "" {
		return fmt.Errorf("MYSQL_URI environment variable is required")
	}

	db, err := mysql.Connect(ctx, mysqlURI)
	if err != nil {
		return fmt.Errorf("connecting to MySQL: %w", err)
	}
	defer func() { _ = db.Close() }()

	dataset := mysql.New(db)

	// Setup Pinecone client for vector similarity
	pineconeAPIKey := os.Getenv("PINECONE_API_KEY")
	pineconeIndexName := os.Getenv("PINECONE_INDEX_NAME")

	if pineconeAPIKey == "" || pineconeIndexName == "" {
		return fmt.Errorf("PINECONE_API_KEY and PINECONE_INDEX_NAME environment variables are required")
	}

	pineconeClient, err := pinecone.NewClient(ctx, pineconeAPIKey, pineconeIndexName)
	if err != nil {
		return fmt.Errorf("connecting to Pinecone: %w", err)
	}

	// Create the cluster update command
	//nolint:gosec // weak random is fine for clustering
	clusterRng := rand.New(rand.NewPCG(0, 0))
	updateClustersCmd := command.NewUpdateUserClusters(
		dataset,
		dataset,
		domain.DefaultClusterConfig(),
		clusterRng,
	)

	// Create the generate recommendations command (actual generation logic)
	generateCmd := command.NewGenerateRecommendations(
		pineconeClient,
		dataset,
		dataset,
		dataset,
		app.DefaultGenerateRecommendationsConfig(),
	)

	// Create the background job runner
	runCmd := command.NewRunRecommendationGeneration(
		updateClustersCmd,
		generateCmd,
		dataset,
		dataset,
		app.DefaultRunRecommendationGenerationConfig(),
	)

	// Execute
	_, err = runCmd.Execute(ctx, command.RunRecommendationGenerationRequest{})
	return err
}
