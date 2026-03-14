package command

import (
	"context"
	"fmt"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// RunRecommendationGenerationRequest is the request for the RunRecommendationGeneration command.
// This command takes no parameters beyond context.
type RunRecommendationGenerationRequest struct{}

// RunRecommendationGenerationConfig holds configuration for background recommendation generation.
type RunRecommendationGenerationConfig struct {
	// CandidateLimit is the number of recommendations to precompute per user.
	// Should be larger than the serving limit to account for read article filtering.
	CandidateLimit int
}

// RunRecommendationGeneration handles background generation of precomputed recommendations.
type RunRecommendationGeneration struct {
	UpdateClustersCmd  *UpdateUserClusters
	GenerateCommand    *GenerateRecommendations
	PrecomputedWriter  datasources.PrecomputedRecommendationWriter
	RegenerationStatus datasources.UserRecommendationRegenerationStatusRepository
	Config             RunRecommendationGenerationConfig
}

// NewRunRecommendationGeneration creates a properly initialized RunRecommendationGeneration command.
func NewRunRecommendationGeneration(
	updateClustersCmd *UpdateUserClusters,
	generateCommand *GenerateRecommendations,
	precomputedWriter datasources.PrecomputedRecommendationWriter,
	regenerationStatus datasources.UserRecommendationRegenerationStatusRepository,
	config RunRecommendationGenerationConfig,
) *RunRecommendationGeneration {
	return &RunRecommendationGeneration{
		UpdateClustersCmd:  updateClustersCmd,
		GenerateCommand:    generateCommand,
		PrecomputedWriter:  precomputedWriter,
		RegenerationStatus: regenerationStatus,
		Config:             config,
	}
}

// Execute runs the background recommendation generation for all users needing regeneration.
func (c *RunRecommendationGeneration) Execute(
	ctx context.Context, _ RunRecommendationGenerationRequest,
) (Empty, error) {
	logger := domain.LoggerFromContext(ctx)

	// Get list of users needing regeneration
	userIDs, err := c.RegenerationStatus.ListUsersNeedingRegeneration(ctx)
	if err != nil {
		return Empty{}, fmt.Errorf("listing users needing regeneration: %w", err)
	}

	if len(userIDs) == 0 {
		logger.InfoContext(ctx, "no users need recommendation regeneration")
		return Empty{}, nil
	}

	logger.InfoContext(ctx, "starting recommendation generation", "user_count", len(userIDs))

	var successCount, failCount int
	for _, userID := range userIDs {
		if err := c.generateForUser(ctx, userID); err != nil {
			logger.ErrorContext(ctx, "failed to generate recommendations for user",
				"user_id", userID, "error", err)
			failCount++
			continue
		}
		successCount++
	}

	logger.InfoContext(ctx, "recommendation generation complete",
		"success_count", successCount, "fail_count", failCount)

	return Empty{}, nil
}

// generateForUser generates and stores recommendations for a single user.
func (c *RunRecommendationGeneration) generateForUser(ctx context.Context, userID string) error {
	logger := domain.LoggerFromContext(ctx)
	logger.DebugContext(ctx, "generating recommendations for user", "user_id", userID)

	// Update interest clusters before generating recommendations
	if _, err := c.UpdateClustersCmd.Execute(ctx, UpdateUserClustersRequest{UserID: userID}); err != nil {
		return fmt.Errorf("updating user clusters: %w", err)
	}

	// Generate recommendations using the generation command
	scoredArticles, err := c.GenerateCommand.Execute(ctx, GenerateRecommendationsRequest{
		UserID: userID,
		Limit:  c.Config.CandidateLimit,
	})
	if err != nil {
		return fmt.Errorf("generating recommendations: %w", err)
	}

	if err := storePrecomputedRecommendations(ctx, c.PrecomputedWriter, c.RegenerationStatus, userID, scoredArticles); err != nil {
		return err
	}

	logger.DebugContext(ctx, "stored recommendations for user",
		"user_id", userID, "count", len(scoredArticles))

	return nil
}
