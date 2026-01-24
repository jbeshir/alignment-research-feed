package domain

import (
	"math"
	"time"
)

// TimestampedVector represents a vector with an associated timestamp.
type TimestampedVector struct {
	Vector    []float32
	Timestamp time.Time
}

// ComputeTemporallyWeightedVector computes a weighted average of vectors using exponential decay.
// More recent vectors have higher weights. The decay follows: weight = exp(-lambda * days_ago)
// where lambda = ln(2) / halfLifeDays.
//
// Returns nil if vectors is empty or all weights sum to zero.
func ComputeTemporallyWeightedVector(
	vectors []TimestampedVector,
	halfLifeDays float64,
	now time.Time,
) []float32 {
	if len(vectors) == 0 {
		return nil
	}

	lambda := math.Ln2 / halfLifeDays

	var weightedSum []float32
	var totalWeight float64

	for _, v := range vectors {
		daysSinceRating := now.Sub(v.Timestamp).Hours() / 24
		weight := math.Exp(-lambda * daysSinceRating)

		if weightedSum == nil {
			weightedSum = make([]float32, len(v.Vector))
		}

		for i, val := range v.Vector {
			weightedSum[i] += float32(weight) * val
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return nil
	}

	// Normalize by total weight
	result := make([]float32, len(weightedSum))
	for i, val := range weightedSum {
		result[i] = val / float32(totalWeight)
	}

	return result
}
