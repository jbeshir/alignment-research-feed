package domain

import (
	"math"
	"math/rand/v2"
)

// ClusterConfig holds configuration for the clustering algorithm.
type ClusterConfig struct {
	// NumClusters is the number of interest clusters to create.
	NumClusters int

	// MinArticlesForClustering is the minimum number of liked articles required.
	// If the user has fewer likes, no clustering is performed.
	MinArticlesForClustering int

	// MaxIterations is the maximum number of k-means iterations.
	MaxIterations int

	// ConvergenceThreshold is the minimum centroid movement to continue iterating.
	ConvergenceThreshold float64
}

// DefaultClusterConfig returns the default clustering configuration.
func DefaultClusterConfig() ClusterConfig {
	return ClusterConfig{
		NumClusters:              3,
		MinArticlesForClustering: 6,
		MaxIterations:            50,
		ConvergenceThreshold:     0.0001,
	}
}

// ClusterResult holds the result of k-means clustering.
type ClusterResult struct {
	Centroids   [][]float32
	Assignments []int
}

// KMeans performs k-means clustering on the given data.
// Returns cluster centroids and assignments (which cluster each point belongs to).
// If data is empty or k is 0, returns nil results.
func KMeans(data [][]float32, k int, config ClusterConfig, rng *rand.Rand) ClusterResult {
	if len(data) == 0 || k == 0 {
		return ClusterResult{}
	}

	// Initialize centroids using k-means++ algorithm
	centroids := initializeCentroidsKMeansPlusPlus(data, k, rng)
	assignments := make([]int, len(data))

	for iter := 0; iter < config.MaxIterations; iter++ {
		// Assignment step: assign each point to nearest centroid
		assignPointsToCentroids(data, centroids, assignments)

		// Update step: recompute centroids
		newCentroids := recomputeCentroids(data, assignments, k, centroids)

		// Check convergence
		if maxCentroidMovement(centroids, newCentroids) < config.ConvergenceThreshold {
			centroids = newCentroids
			break
		}

		centroids = newCentroids
	}

	return ClusterResult{
		Centroids:   centroids,
		Assignments: assignments,
	}
}

// assignPointsToCentroids assigns each data point to its nearest centroid.
func assignPointsToCentroids(data [][]float32, centroids [][]float32, assignments []int) {
	for i, point := range data {
		assignments[i] = findNearestCentroid(point, centroids)
	}
}

// recomputeCentroids calculates new centroid positions based on current assignments.
func recomputeCentroids(
	data [][]float32,
	assignments []int,
	k int,
	oldCentroids [][]float32,
) [][]float32 {
	dim := len(data[0])
	newCentroids := make([][]float32, k)
	counts := make([]int, k)

	for i := range newCentroids {
		newCentroids[i] = make([]float32, dim)
	}

	// Sum up points for each cluster
	for i, point := range data {
		cluster := assignments[i]
		counts[cluster]++
		for j, val := range point {
			newCentroids[cluster][j] += val
		}
	}

	// Normalize centroids (or keep old centroid for empty clusters)
	for i := range newCentroids {
		if counts[i] > 0 {
			for j := range newCentroids[i] {
				newCentroids[i][j] /= float32(counts[i])
			}
		} else {
			newCentroids[i] = oldCentroids[i]
		}
	}

	return newCentroids
}

// maxCentroidMovement returns the maximum distance any centroid moved.
func maxCentroidMovement(oldCentroids, newCentroids [][]float32) float64 {
	maxMovement := float64(0)
	for i := range oldCentroids {
		movement := EuclideanDistance(oldCentroids[i], newCentroids[i])
		if movement > maxMovement {
			maxMovement = movement
		}
	}
	return maxMovement
}

// initializeCentroidsKMeansPlusPlus initializes centroids using k-means++ algorithm.
func initializeCentroidsKMeansPlusPlus(data [][]float32, k int, rng *rand.Rand) [][]float32 {
	n := len(data)
	dim := len(data[0])
	centroids := make([][]float32, k)

	// Choose first centroid randomly
	firstIdx := rng.IntN(n)
	centroids[0] = make([]float32, dim)
	copy(centroids[0], data[firstIdx])

	// Choose remaining centroids with probability proportional to distance squared
	distances := make([]float64, n)
	for i := 1; i < k; i++ {
		// Compute distances to nearest existing centroid
		totalDist := float64(0)
		for j, point := range data {
			minDist := math.MaxFloat64
			for ci := 0; ci < i; ci++ {
				dist := EuclideanDistanceSquared(point, centroids[ci])
				if dist < minDist {
					minDist = dist
				}
			}
			distances[j] = minDist
			totalDist += minDist
		}

		// Choose next centroid with probability proportional to distance squared
		target := rng.Float64() * totalDist
		cumulative := float64(0)
		chosenIdx := 0
		for j, d := range distances {
			cumulative += d
			if cumulative >= target {
				chosenIdx = j
				break
			}
		}

		centroids[i] = make([]float32, dim)
		copy(centroids[i], data[chosenIdx])
	}

	return centroids
}

// findNearestCentroid finds the index of the nearest centroid to the given point.
func findNearestCentroid(point []float32, centroids [][]float32) int {
	minDist := math.MaxFloat64
	minIdx := 0

	for i, centroid := range centroids {
		dist := EuclideanDistanceSquared(point, centroid)
		if dist < minDist {
			minDist = dist
			minIdx = i
		}
	}

	return minIdx
}

// EuclideanDistance computes the Euclidean distance between two vectors.
func EuclideanDistance(a, b []float32) float64 {
	return math.Sqrt(EuclideanDistanceSquared(a, b))
}

// EuclideanDistanceSquared computes the squared Euclidean distance between two vectors.
func EuclideanDistanceSquared(a, b []float32) float64 {
	sum := float64(0)
	for i := range a {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}
	return sum
}

// CountClusterAssignments counts how many points are assigned to each cluster.
func CountClusterAssignments(assignments []int, k int) []int {
	counts := make([]int, k)
	for _, a := range assignments {
		if a >= 0 && a < k {
			counts[a]++
		}
	}
	return counts
}
