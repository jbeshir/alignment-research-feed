package command

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
)

// UpdateUserClustersRequest is the request for the UpdateUserClusters command.
type UpdateUserClustersRequest struct {
	UserID string
}

// UpdateUserClusters recomputes interest clusters for a user based on their liked articles.
type UpdateUserClusters struct {
	VectorsGetter datasources.UserArticleVectorsGetter
	ClusterWriter datasources.UserInterestClusterWriter
	Config        domain.ClusterConfig
	Rand          *rand.Rand
}

// NewUpdateUserClusters creates a properly initialized UpdateUserClusters command.
func NewUpdateUserClusters(
	vectorsGetter datasources.UserArticleVectorsGetter,
	clusterWriter datasources.UserInterestClusterWriter,
	config domain.ClusterConfig,
	rng *rand.Rand,
) *UpdateUserClusters {
	return &UpdateUserClusters{
		VectorsGetter: vectorsGetter,
		ClusterWriter: clusterWriter,
		Config:        config,
		Rand:          rng,
	}
}

// Execute runs k-means clustering on the user's liked article vectors
// and stores the resulting cluster centroids.
func (c *UpdateUserClusters) Execute(ctx context.Context, req UpdateUserClustersRequest) (Empty, error) {
	logger := domain.LoggerFromContext(ctx)

	// Get user's thumbs up vectors
	vectors, err := c.VectorsGetter.GetUserArticleVectorsByType(ctx, req.UserID, domain.RatingTypeThumbsUp)
	if err != nil {
		return Empty{}, fmt.Errorf("getting thumbs up vectors: %w", err)
	}

	// Check minimum articles requirement
	if len(vectors) < c.Config.MinArticlesForClustering {
		logger.DebugContext(ctx, "not enough articles for clustering",
			"count", len(vectors), "min", c.Config.MinArticlesForClustering)
		// Clear existing clusters since we don't have enough data
		if err := c.ClusterWriter.DeleteUserInterestClusters(ctx, req.UserID); err != nil {
			logger.WarnContext(ctx, "failed to delete user clusters", "error", err)
		}
		return Empty{}, nil
	}

	// Extract just the vectors for clustering
	data := make([][]float32, len(vectors))
	for i, v := range vectors {
		data[i] = v.Vector
	}

	// Determine actual number of clusters (can't have more clusters than data points)
	k := c.Config.NumClusters
	if k > len(data) {
		k = len(data)
	}

	// Run k-means clustering
	result := domain.KMeans(data, k, c.Config, c.Rand)

	// Count articles per cluster
	clusterCounts := domain.CountClusterAssignments(result.Assignments, k)

	// Delete existing clusters and save new ones
	if err := c.ClusterWriter.DeleteUserInterestClusters(ctx, req.UserID); err != nil {
		return Empty{}, fmt.Errorf("deleting old clusters: %w", err)
	}

	for i, centroid := range result.Centroids {
		if clusterCounts[i] == 0 {
			continue // Skip empty clusters
		}
		if err := c.ClusterWriter.UpsertUserInterestCluster(
			ctx, req.UserID, i, centroid, clusterCounts[i],
		); err != nil {
			return Empty{}, fmt.Errorf("saving cluster %d: %w", i, err)
		}
	}

	logger.DebugContext(ctx, "updated user clusters",
		"numClusters", k, "clusterCounts", clusterCounts)

	return Empty{}, nil
}
