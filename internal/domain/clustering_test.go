package domain

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRand creates a deterministic random number generator for testing.
func newTestRand(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed)) //nolint:gosec // weak random is fine for clustering tests
}

func TestEuclideanDistanceSquared(t *testing.T) {
	cases := []struct {
		name string
		a    []float32
		b    []float32
		want float64
	}{
		{
			name: "same_point",
			a:    []float32{1.0, 2.0, 3.0},
			b:    []float32{1.0, 2.0, 3.0},
			want: 0.0,
		},
		{
			name: "unit_distance",
			a:    []float32{0.0, 0.0},
			b:    []float32{1.0, 0.0},
			want: 1.0,
		},
		{
			name: "diagonal",
			a:    []float32{0.0, 0.0},
			b:    []float32{3.0, 4.0},
			want: 25.0, // 3^2 + 4^2
		},
		{
			name: "negative_values",
			a:    []float32{-1.0, -1.0},
			b:    []float32{1.0, 1.0},
			want: 8.0, // 2^2 + 2^2
		},
		{
			name: "empty_vectors",
			a:    []float32{},
			b:    []float32{},
			want: 0.0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EuclideanDistanceSquared(tc.a, tc.b)
			assert.InDelta(t, tc.want, got, 0.0001)
		})
	}
}

func TestEuclideanDistance(t *testing.T) {
	cases := []struct {
		name string
		a    []float32
		b    []float32
		want float64
	}{
		{
			name: "same_point",
			a:    []float32{1.0, 2.0, 3.0},
			b:    []float32{1.0, 2.0, 3.0},
			want: 0.0,
		},
		{
			name: "unit_distance",
			a:    []float32{0.0, 0.0},
			b:    []float32{1.0, 0.0},
			want: 1.0,
		},
		{
			name: "diagonal_3_4_5",
			a:    []float32{0.0, 0.0},
			b:    []float32{3.0, 4.0},
			want: 5.0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EuclideanDistance(tc.a, tc.b)
			assert.InDelta(t, tc.want, got, 0.0001)
		})
	}
}

func TestKMeans_EmptyData(t *testing.T) {
	config := DefaultClusterConfig()
	rng := newTestRand(42)

	result := KMeans(nil, 3, config, rng)
	assert.Nil(t, result.Centroids)
	assert.Nil(t, result.Assignments)

	result = KMeans([][]float32{}, 3, config, rng)
	assert.Nil(t, result.Centroids)
	assert.Nil(t, result.Assignments)
}

func TestKMeans_ZeroClusters(t *testing.T) {
	config := DefaultClusterConfig()
	rng := newTestRand(42)
	data := [][]float32{{1.0, 2.0}, {3.0, 4.0}}

	result := KMeans(data, 0, config, rng)
	assert.Nil(t, result.Centroids)
	assert.Nil(t, result.Assignments)
}

func TestKMeans_SinglePoint(t *testing.T) {
	config := DefaultClusterConfig()
	rng := newTestRand(42)
	data := [][]float32{{1.0, 2.0}}

	result := KMeans(data, 1, config, rng)

	require.Len(t, result.Centroids, 1)
	assert.InDelta(t, 1.0, result.Centroids[0][0], 0.0001)
	assert.InDelta(t, 2.0, result.Centroids[0][1], 0.0001)
	assert.Equal(t, []int{0}, result.Assignments)
}

func TestKMeans_TwoDistinctClusters(t *testing.T) {
	config := ClusterConfig{
		NumClusters:          2,
		MaxIterations:        50,
		ConvergenceThreshold: 0.0001,
	}

	// Two clearly separated clusters:
	// Cluster 1: points around (0, 0)
	// Cluster 2: points around (10, 10)
	data := [][]float32{
		{0.0, 0.0},
		{0.1, 0.1},
		{-0.1, 0.1},
		{0.1, -0.1},
		{10.0, 10.0},
		{10.1, 10.1},
		{9.9, 10.1},
		{10.1, 9.9},
	}

	rng := newTestRand(42)
	result := KMeans(data, 2, config, rng)

	require.Len(t, result.Centroids, 2)
	require.Len(t, result.Assignments, 8)

	// Verify that points are correctly clustered
	// First 4 points should be in one cluster, last 4 in another
	cluster0 := result.Assignments[0]
	for i := 0; i < 4; i++ {
		assert.Equal(t, cluster0, result.Assignments[i], "point %d should be in same cluster as point 0", i)
	}

	cluster1 := result.Assignments[4]
	assert.NotEqual(t, cluster0, cluster1, "clusters should be different")
	for i := 4; i < 8; i++ {
		assert.Equal(t, cluster1, result.Assignments[i], "point %d should be in same cluster as point 4", i)
	}

	// Verify centroids are near expected positions
	centroid0 := result.Centroids[cluster0]
	centroid1 := result.Centroids[cluster1]

	dist0 := EuclideanDistance(centroid0, []float32{0.025, 0.025})
	dist1 := EuclideanDistance(centroid1, []float32{10.025, 10.025})
	assert.Less(t, dist0, 0.5, "centroid 0 should be near (0,0)")
	assert.Less(t, dist1, 0.5, "centroid 1 should be near (10,10)")
}

func TestKMeans_Convergence(t *testing.T) {
	// Test that k-means converges and doesn't run forever
	config := ClusterConfig{
		NumClusters:          2,
		MaxIterations:        100,
		ConvergenceThreshold: 0.0001,
	}

	data := [][]float32{
		{0.0, 0.0},
		{1.0, 0.0},
		{0.0, 1.0},
		{10.0, 10.0},
		{11.0, 10.0},
		{10.0, 11.0},
	}

	rng := newTestRand(42)
	result := KMeans(data, 2, config, rng)

	// Just verify it completes and returns valid results
	require.Len(t, result.Centroids, 2)
	require.Len(t, result.Assignments, 6)
}

func TestKMeans_MoreClustersThanPoints(t *testing.T) {
	config := DefaultClusterConfig()
	data := [][]float32{
		{0.0, 0.0},
		{1.0, 1.0},
	}

	// Request 5 clusters for 2 points - should still work
	rng := newTestRand(42)
	result := KMeans(data, 5, config, rng)

	// Should return 5 centroids, but only 2 will have assignments
	require.Len(t, result.Centroids, 5)
	require.Len(t, result.Assignments, 2)
}

func TestKMeans_SingleCluster(t *testing.T) {
	config := ClusterConfig{
		NumClusters:          1,
		MaxIterations:        50,
		ConvergenceThreshold: 0.0001,
	}

	data := [][]float32{
		{0.0, 0.0},
		{2.0, 0.0},
		{0.0, 2.0},
		{2.0, 2.0},
	}

	rng := newTestRand(42)
	result := KMeans(data, 1, config, rng)

	require.Len(t, result.Centroids, 1)
	require.Len(t, result.Assignments, 4)

	// All points should be in cluster 0
	for i, a := range result.Assignments {
		assert.Equal(t, 0, a, "point %d should be in cluster 0", i)
	}

	// Centroid should be at the mean: (1, 1)
	assert.InDelta(t, 1.0, result.Centroids[0][0], 0.0001)
	assert.InDelta(t, 1.0, result.Centroids[0][1], 0.0001)
}

func TestKMeans_ThreeDimensional(t *testing.T) {
	config := ClusterConfig{
		NumClusters:          2,
		MaxIterations:        50,
		ConvergenceThreshold: 0.0001,
	}

	// Two clusters in 3D space
	data := [][]float32{
		{0.0, 0.0, 0.0},
		{0.1, 0.1, 0.1},
		{100.0, 100.0, 100.0},
		{100.1, 100.1, 100.1},
	}

	rng := newTestRand(42)
	result := KMeans(data, 2, config, rng)

	require.Len(t, result.Centroids, 2)
	require.Len(t, result.Assignments, 4)

	// First two points should be in same cluster
	assert.Equal(t, result.Assignments[0], result.Assignments[1])
	// Last two points should be in same cluster
	assert.Equal(t, result.Assignments[2], result.Assignments[3])
	// Different clusters
	assert.NotEqual(t, result.Assignments[0], result.Assignments[2])
}

func TestCountClusterAssignments(t *testing.T) {
	cases := []struct {
		name        string
		assignments []int
		k           int
		want        []int
	}{
		{
			name:        "balanced_two_clusters",
			assignments: []int{0, 1, 0, 1, 0, 1},
			k:           2,
			want:        []int{3, 3},
		},
		{
			name:        "unbalanced",
			assignments: []int{0, 0, 0, 1, 2, 2},
			k:           3,
			want:        []int{3, 1, 2},
		},
		{
			name:        "empty_assignments",
			assignments: []int{},
			k:           3,
			want:        []int{0, 0, 0},
		},
		{
			name:        "single_cluster",
			assignments: []int{0, 0, 0, 0},
			k:           1,
			want:        []int{4},
		},
		{
			name:        "with_empty_clusters",
			assignments: []int{0, 0, 2, 2},
			k:           3,
			want:        []int{2, 0, 2},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CountClusterAssignments(tc.assignments, tc.k)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDefaultClusterConfig(t *testing.T) {
	config := DefaultClusterConfig()

	assert.Equal(t, 3, config.NumClusters)
	assert.Equal(t, 6, config.MinArticlesForClustering)
	assert.Equal(t, 50, config.MaxIterations)
	assert.InDelta(t, 0.0001, config.ConvergenceThreshold, 0.00001)
}

func TestKMeans_Determinism(t *testing.T) {
	// With the same seed, results should be identical
	config := ClusterConfig{
		NumClusters:          2,
		MaxIterations:        50,
		ConvergenceThreshold: 0.0001,
	}

	data := [][]float32{
		{0.0, 0.0},
		{1.0, 1.0},
		{10.0, 10.0},
		{11.0, 11.0},
	}

	rng1 := newTestRand(12345)
	rng2 := newTestRand(12345)

	result1 := KMeans(data, 2, config, rng1)
	result2 := KMeans(data, 2, config, rng2)

	assert.Equal(t, result1.Assignments, result2.Assignments)
	for i := range result1.Centroids {
		for j := range result1.Centroids[i] {
			assert.InDelta(t, result1.Centroids[i][j], result2.Centroids[i][j], 0.0001)
		}
	}
}

func TestKMeans_HighDimensional(t *testing.T) {
	// Test with higher dimensional vectors (similar to embedding vectors)
	config := ClusterConfig{
		NumClusters:          2,
		MaxIterations:        50,
		ConvergenceThreshold: 0.0001,
	}

	dim := 128
	data := make([][]float32, 4)

	// Create two clusters: one near origin, one far away
	for i := 0; i < 2; i++ {
		data[i] = make([]float32, dim)
		for j := 0; j < dim; j++ {
			data[i][j] = float32(j) * 0.01 //nolint:gosec // j is bounded by dim
		}
	}
	for i := 2; i < 4; i++ {
		data[i] = make([]float32, dim)
		for j := 0; j < dim; j++ {
			data[i][j] = float32(j)*0.01 + 100.0 //nolint:gosec // j is bounded by dim
		}
	}

	rng := newTestRand(42)
	result := KMeans(data, 2, config, rng)

	require.Len(t, result.Centroids, 2)
	require.Len(t, result.Assignments, 4)

	// First two should be in same cluster, last two in another
	assert.Equal(t, result.Assignments[0], result.Assignments[1])
	assert.Equal(t, result.Assignments[2], result.Assignments[3])
	assert.NotEqual(t, result.Assignments[0], result.Assignments[2])
}

func TestKMeans_EmptyClusterHandling(t *testing.T) {
	// Test that empty clusters don't cause issues
	config := ClusterConfig{
		NumClusters:          3,
		MaxIterations:        50,
		ConvergenceThreshold: 0.0001,
	}

	// All points very close together - likely only one cluster will have members
	data := [][]float32{
		{0.0, 0.0},
		{0.001, 0.001},
		{0.002, 0.002},
	}

	rng := newTestRand(42)

	// Should not panic even if some clusters become empty
	result := KMeans(data, 3, config, rng)

	require.Len(t, result.Centroids, 3)
	require.Len(t, result.Assignments, 3)
}

func TestFindNearestCentroid(t *testing.T) {
	centroids := [][]float32{
		{0.0, 0.0},
		{10.0, 10.0},
		{-10.0, -10.0},
	}

	cases := []struct {
		name  string
		point []float32
		want  int
	}{
		{
			name:  "nearest_to_origin",
			point: []float32{0.1, 0.1},
			want:  0,
		},
		{
			name:  "nearest_to_positive",
			point: []float32{9.0, 9.0},
			want:  1,
		},
		{
			name:  "nearest_to_negative",
			point: []float32{-8.0, -8.0},
			want:  2,
		},
		{
			name:  "exactly_on_centroid",
			point: []float32{10.0, 10.0},
			want:  1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findNearestCentroid(tc.point, centroids)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestKMeans_MaxIterationsRespected(t *testing.T) {
	// Create a scenario where convergence is slow
	config := ClusterConfig{
		NumClusters:          2,
		MaxIterations:        2,   // Very low
		ConvergenceThreshold: 0.0, // Never converge by threshold
	}

	data := [][]float32{
		{0.0, 0.0},
		{1.0, 1.0},
		{10.0, 10.0},
		{11.0, 11.0},
	}

	rng := newTestRand(42)

	// Should complete without infinite loop
	result := KMeans(data, 2, config, rng)

	require.Len(t, result.Centroids, 2)
	require.Len(t, result.Assignments, 4)
}

func BenchmarkKMeans(b *testing.B) {
	config := ClusterConfig{
		NumClusters:          5,
		MaxIterations:        50,
		ConvergenceThreshold: 0.0001,
	}

	// Generate test data
	data := make([][]float32, 100)
	for i := range data {
		data[i] = make([]float32, 128)
		for j := range data[i] {
			data[i][j] = float32(i*128+j) * 0.001
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rng := newTestRand(42)
		KMeans(data, 5, config, rng)
	}
}

func BenchmarkEuclideanDistanceSquared(b *testing.B) {
	a := make([]float32, 128)
	c := make([]float32, 128)
	for i := range a {
		a[i] = float32(i) * 0.01
		c[i] = float32(i) * 0.02
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EuclideanDistanceSquared(a, c)
	}
}

func TestKMeansPlusPlusInitialization(t *testing.T) {
	// Test that k-means++ initialization spreads centroids well
	data := [][]float32{
		{0.0, 0.0},   // Cluster 1
		{0.1, 0.0},   // Cluster 1
		{50.0, 50.0}, // Cluster 2
		{50.1, 50.0}, // Cluster 2
		{100.0, 0.0}, // Cluster 3
		{100.1, 0.0}, // Cluster 3
	}

	rng := newTestRand(42)
	centroids := initializeCentroidsKMeansPlusPlus(data, 3, rng)

	require.Len(t, centroids, 3)

	// Calculate total spread - centroids should be reasonably spread apart
	totalDist := 0.0
	for i := 0; i < len(centroids); i++ {
		for j := i + 1; j < len(centroids); j++ {
			totalDist += EuclideanDistance(centroids[i], centroids[j])
		}
	}

	// Centroids should be spread (not all clustered together)
	assert.Greater(t, totalDist, 50.0, "centroids should be spread apart")
}

func TestKMeans_NaNHandling(t *testing.T) {
	// Ensure algorithm handles edge cases without producing NaN
	config := ClusterConfig{
		NumClusters:          2,
		MaxIterations:        10,
		ConvergenceThreshold: 0.0001,
	}

	// Points that could cause division issues
	data := [][]float32{
		{0.0, 0.0},
		{0.0, 0.0}, // Duplicate point
		{1.0, 1.0},
		{1.0, 1.0}, // Duplicate point
	}

	rng := newTestRand(42)
	result := KMeans(data, 2, config, rng)

	// Verify no NaN in centroids
	for i, centroid := range result.Centroids {
		for j, v := range centroid {
			assert.False(t, math.IsNaN(float64(v)), "centroid[%d][%d] should not be NaN", i, j)
			assert.False(t, math.IsInf(float64(v), 0), "centroid[%d][%d] should not be Inf", i, j)
		}
	}
}
