package app

import (
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/command"
)

// DefaultGenerateRecommendationsConfig returns the default config for recommendation generation.
func DefaultGenerateRecommendationsConfig() command.GenerateRecommendationsConfig {
	return command.GenerateRecommendationsConfig{
		TemporalDecayHalfLifeDays: 90,
		NegativeSignalWeight:      0.3,
		UseInterestClusters:       true,
		CandidatesPerCluster:      20,
	}
}

// DefaultRecommendArticlesConfig returns the default config for serving recommendations.
func DefaultRecommendArticlesConfig() command.RecommendArticlesConfig {
	return command.RecommendArticlesConfig{
		PrecomputedStaleThreshold: 48 * time.Hour,
		PrecomputedFetchLimit:     200,
	}
}

// DefaultRunRecommendationGenerationConfig returns the default config for background generation.
func DefaultRunRecommendationGenerationConfig() command.RunRecommendationGenerationConfig {
	return command.RunRecommendationGenerationConfig{
		CandidateLimit: 200,
	}
}
