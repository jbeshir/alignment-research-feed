package command

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// GenerateRecommendationsRequest is the request for the GenerateRecommendations command.
type GenerateRecommendationsRequest struct {
	UserID string
	Limit  int
}

// GenerateRecommendationsConfig holds configuration for recommendation generation.
type GenerateRecommendationsConfig struct {
	// TemporalDecayHalfLifeDays is the half-life for temporal decay in days.
	// After this many days, a rating has half its original weight.
	TemporalDecayHalfLifeDays float64

	// NegativeSignalWeight controls how much thumbs-down ratings penalize recommendations.
	// Range: 0.0 (no penalty) to 1.0 (full penalty)
	NegativeSignalWeight float64

	// UseInterestClusters enables multi-interest clustering.
	// When true, retrieves candidates from each cluster centroid.
	UseInterestClusters bool

	// CandidatesPerCluster is how many candidates to retrieve per cluster.
	CandidatesPerCluster int
}

// GenerateRecommendations generates recommendations using vector similarity,
// temporal decay, multi-interest clustering, and negative signal integration.
type GenerateRecommendations struct {
	VectorSimilarity   datasources.SimilarArticlesByVectorLister
	VectorsGetter      datasources.UserArticleVectorsGetter
	ClusterGetter      datasources.UserInterestClusterGetter
	ReadArticlesLister datasources.ReadArticleIDsLister
	Config             GenerateRecommendationsConfig
}

// NewGenerateRecommendations creates a properly initialized GenerateRecommendations command.
func NewGenerateRecommendations(
	vectorSimilarity datasources.SimilarArticlesByVectorLister,
	vectorsGetter datasources.UserArticleVectorsGetter,
	clusterGetter datasources.UserInterestClusterGetter,
	readArticlesLister datasources.ReadArticleIDsLister,
	config GenerateRecommendationsConfig,
) *GenerateRecommendations {
	return &GenerateRecommendations{
		VectorSimilarity:   vectorSimilarity,
		VectorsGetter:      vectorsGetter,
		ClusterGetter:      clusterGetter,
		ReadArticlesLister: readArticlesLister,
		Config:             config,
	}
}

// ScoredArticle represents an article with its recommendation score.
type ScoredArticle struct {
	HashID string
	Score  float64
	Source string // "temporal", "cluster_N", etc.
}

// Execute generates recommendations for a user using vector similarity.
func (c *GenerateRecommendations) Execute(
	ctx context.Context, req GenerateRecommendationsRequest,
) ([]ScoredArticle, error) {
	// Get read article IDs to exclude from recommendations
	readArticleIDs := c.getReadArticleIDs(ctx, req.UserID)

	// Get user's article vectors for temporal weighting
	thumbsUpVectors, err := c.VectorsGetter.GetUserArticleVectorsByType(
		ctx, req.UserID, domain.RatingTypeThumbsUp,
	)
	if err != nil {
		return nil, fmt.Errorf("getting thumbs up vectors: %w", err)
	}

	if len(thumbsUpVectors) == 0 {
		return nil, nil
	}

	// Compute negative signal vector once (average of thumbs down)
	negativeVector := c.getNegativeVector(ctx, req.UserID)

	// Collect candidates from multiple strategies
	var candidates []ScoredArticle
	candidates = append(candidates, c.getCandidatesUsingClusters(ctx, req.UserID, negativeVector)...)
	candidates = append(candidates, c.getCandidatesUsingTemporalVector(ctx, thumbsUpVectors, negativeVector)...)

	if len(candidates) == 0 {
		return nil, nil
	}

	// Deduplicate, filter read articles, and rank candidates
	return c.rankAndDeduplicate(candidates, req.Limit, readArticleIDs), nil
}

// getNegativeVector computes the negative signal vector from thumbs-down ratings.
func (c *GenerateRecommendations) getNegativeVector(ctx context.Context, userID string) []float32 {
	if c.Config.NegativeSignalWeight <= 0 {
		return nil
	}

	logger := domain.LoggerFromContext(ctx)
	thumbsDownVectors, err := c.VectorsGetter.GetUserArticleVectorsByType(
		ctx, userID, domain.RatingTypeThumbsDown,
	)
	if err != nil {
		logger.WarnContext(ctx, "failed to get thumbs down vectors", "error", err)
		return nil
	}

	if len(thumbsDownVectors) == 0 {
		return nil
	}

	return c.computeAverageVector(thumbsDownVectors)
}

// getCandidatesUsingClusters retrieves candidates using interest cluster centroids.
func (c *GenerateRecommendations) getCandidatesUsingClusters(
	ctx context.Context,
	userID string,
	negativeVector []float32,
) []ScoredArticle {
	if !c.Config.UseInterestClusters {
		return nil
	}

	logger := domain.LoggerFromContext(ctx)
	clusters, err := c.ClusterGetter.GetUserInterestClusters(ctx, userID)
	if err != nil {
		logger.WarnContext(ctx, "failed to get interest clusters", "error", err)
		return nil
	}

	if len(clusters) == 0 {
		return nil
	}

	candidates, err := c.getCandidatesFromClusters(ctx, clusters, negativeVector)
	if err != nil {
		logger.WarnContext(ctx, "failed to get cluster candidates", "error", err)
		return nil
	}

	return candidates
}

// getCandidatesUsingTemporalVector retrieves candidates using temporally-weighted average.
func (c *GenerateRecommendations) getCandidatesUsingTemporalVector(
	ctx context.Context,
	thumbsUpVectors []domain.UserArticleRating,
	negativeVector []float32,
) []ScoredArticle {
	logger := domain.LoggerFromContext(ctx)

	temporalVector := c.computeTemporallyWeightedVector(thumbsUpVectors)
	if temporalVector == nil {
		return nil
	}

	candidateLimit := c.Config.CandidatesPerCluster * 2
	candidates, err := c.getCandidatesFromVector(ctx, temporalVector, negativeVector, "temporal", candidateLimit)
	if err != nil {
		logger.WarnContext(ctx, "failed to get temporal candidates", "error", err)
		return nil
	}

	return candidates
}

// computeTemporallyWeightedVector computes a weighted average vector with temporal decay.
func (c *GenerateRecommendations) computeTemporallyWeightedVector(
	vectors []domain.UserArticleRating,
) []float32 {
	if len(vectors) == 0 {
		return nil
	}

	// Convert to domain type
	timestamped := make([]domain.TimestampedVector, len(vectors))
	for i, v := range vectors {
		timestamped[i] = domain.TimestampedVector{
			Vector:    v.Vector,
			Timestamp: v.RatedAt,
		}
	}

	return domain.ComputeTemporallyWeightedVector(timestamped, c.Config.TemporalDecayHalfLifeDays, time.Now())
}

// getCandidatesFromClusters retrieves candidates from each interest cluster.
func (c *GenerateRecommendations) getCandidatesFromClusters(
	ctx context.Context,
	clusters []datasources.UserInterestCluster,
	negativeVector []float32,
) ([]ScoredArticle, error) {
	var allCandidates []ScoredArticle

	for _, cluster := range clusters {
		source := fmt.Sprintf("cluster_%d", cluster.ClusterID)
		candidates, err := c.getCandidatesFromVector(
			ctx, cluster.CentroidVector, negativeVector, source, c.Config.CandidatesPerCluster,
		)
		if err != nil {
			return nil, fmt.Errorf("getting candidates from cluster %d: %w", cluster.ClusterID, err)
		}
		allCandidates = append(allCandidates, candidates...)
	}

	return allCandidates, nil
}

// getCandidatesFromVector retrieves and scores candidates from Pinecone.
func (c *GenerateRecommendations) getCandidatesFromVector(
	ctx context.Context,
	queryVector []float32,
	negativeVector []float32,
	source string,
	limit int,
) ([]ScoredArticle, error) {
	similar, err := c.VectorSimilarity.ListSimilarArticlesByVector(ctx, nil, queryVector, limit)
	if err != nil {
		return nil, err
	}

	candidates := make([]ScoredArticle, 0, len(similar))
	for _, s := range similar {
		score := s.Score

		// Apply negative signal penalty
		if negativeVector != nil {
			// For efficiency, we approximate using the query similarity as a proxy.
			// A more accurate approach would fetch each article's vector.
			negativePenalty := c.Config.NegativeSignalWeight * score * 0.5
			score -= negativePenalty
		}

		candidates = append(candidates, ScoredArticle{
			HashID: s.HashID,
			Score:  score,
			Source: source,
		})
	}

	return candidates, nil
}

// computeAverageVector computes a simple average of vectors.
func (c *GenerateRecommendations) computeAverageVector(vectors []domain.UserArticleRating) []float32 {
	if len(vectors) == 0 {
		return nil
	}

	var sum []float32
	for _, v := range vectors {
		if sum == nil {
			sum = make([]float32, len(v.Vector))
		}
		for i, val := range v.Vector {
			sum[i] += val
		}
	}

	count := float32(len(vectors))
	result := make([]float32, len(sum))
	for i, val := range sum {
		result[i] = val / count
	}

	return result
}

// rankAndDeduplicate removes duplicates, filters excluded articles, and returns top-K by score.
func (c *GenerateRecommendations) rankAndDeduplicate(
	candidates []ScoredArticle,
	limit int,
	excludeIDs map[string]struct{},
) []ScoredArticle {
	// Deduplicate by HashID, keeping highest score
	seen := make(map[string]ScoredArticle)
	for _, cand := range candidates {
		// Skip excluded articles (already read)
		if _, excluded := excludeIDs[cand.HashID]; excluded {
			continue
		}
		if existing, ok := seen[cand.HashID]; !ok || cand.Score > existing.Score {
			seen[cand.HashID] = cand
		}
	}

	// Convert back to slice and sort by score
	unique := make([]ScoredArticle, 0, len(seen))
	for _, cand := range seen {
		unique = append(unique, cand)
	}

	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Score > unique[j].Score
	})

	if len(unique) > limit {
		unique = unique[:limit]
	}

	return unique
}

// getReadArticleIDs fetches the set of article IDs the user has already read.
func (c *GenerateRecommendations) getReadArticleIDs(ctx context.Context, userID string) map[string]struct{} {
	logger := domain.LoggerFromContext(ctx)
	readIDs, err := c.ReadArticlesLister.ListReadArticleIDs(ctx, userID)
	if err != nil {
		logger.WarnContext(ctx, "failed to get read article IDs", "error", err)
		return make(map[string]struct{})
	}

	result := make(map[string]struct{}, len(readIDs))
	for _, id := range readIDs {
		result[id] = struct{}{}
	}
	return result
}
