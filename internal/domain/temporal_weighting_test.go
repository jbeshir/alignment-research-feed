package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeTemporallyWeightedVector(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	halfLifeDays := 30.0

	cases := []struct {
		name     string
		vectors  []TimestampedVector
		expected []float32
	}{
		{
			name:     "empty_vectors_returns_nil",
			vectors:  nil,
			expected: nil,
		},
		{
			name:     "empty_slice_returns_nil",
			vectors:  []TimestampedVector{},
			expected: nil,
		},
		{
			name: "single_vector_returns_itself",
			vectors: []TimestampedVector{
				{Vector: []float32{1.0, 2.0, 3.0}, Timestamp: now},
			},
			expected: []float32{1.0, 2.0, 3.0},
		},
		{
			name: "two_equal_age_vectors_averaged",
			vectors: []TimestampedVector{
				{Vector: []float32{1.0, 0.0}, Timestamp: now},
				{Vector: []float32{0.0, 1.0}, Timestamp: now},
			},
			expected: []float32{0.5, 0.5},
		},
		{
			name: "recent_vector_weighted_higher",
			vectors: []TimestampedVector{
				{Vector: []float32{1.0, 0.0}, Timestamp: now},                    // today, weight ~1.0
				{Vector: []float32{0.0, 1.0}, Timestamp: now.AddDate(0, 0, -30)}, // 30 days ago (half-life), weight ~0.5
			},
			// weights: 1.0 and 0.5, total = 1.5
			// result: [1.0*1.0 + 0.0*0.5, 0.0*1.0 + 1.0*0.5] / 1.5 = [1.0, 0.5] / 1.5 = [0.667, 0.333]
			expected: []float32{0.667, 0.333},
		},
		{
			name: "very_old_vector_has_low_weight",
			vectors: []TimestampedVector{
				{Vector: []float32{1.0, 0.0}, Timestamp: now},                    // today
				{Vector: []float32{0.0, 1.0}, Timestamp: now.AddDate(0, 0, -90)}, // 90 days ago (3 half-lives), weight ~0.125
			},
			// weights: 1.0 and 0.125, total = 1.125
			// result: [1.0, 0.125] / 1.125 = [0.889, 0.111]
			expected: []float32{0.889, 0.111},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ComputeTemporallyWeightedVector(tc.vectors, halfLifeDays, now)

			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			require.Len(t, result, len(tc.expected))

			for i := range tc.expected {
				assert.InDelta(t, tc.expected[i], result[i], 0.01,
					"mismatch at index %d: expected %f, got %f", i, tc.expected[i], result[i])
			}
		})
	}
}

func TestComputeTemporallyWeightedVector_DifferentHalfLives(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	vectors := []TimestampedVector{
		{Vector: []float32{1.0}, Timestamp: now},
		{Vector: []float32{0.0}, Timestamp: now.AddDate(0, 0, -10)}, // 10 days ago
	}

	// With 10-day half-life, the older vector has weight 0.5
	result10 := ComputeTemporallyWeightedVector(vectors, 10, now)
	// weights: 1.0 and 0.5, total = 1.5
	// result: [1.0*1.0 + 0.0*0.5] / 1.5 = 0.667
	assert.InDelta(t, 0.667, result10[0], 0.01)

	// With 100-day half-life, the older vector has weight ~0.93
	result100 := ComputeTemporallyWeightedVector(vectors, 100, now)
	// weights: 1.0 and ~0.933, total = ~1.933
	// result: [1.0*1.0 + 0.0*0.933] / 1.933 = 0.517
	assert.InDelta(t, 0.517, result100[0], 0.01)
}

func TestComputeTemporallyWeightedVector_FutureTimestamp(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	// Future timestamps get weight > 1, which is fine mathematically
	vectors := []TimestampedVector{
		{Vector: []float32{1.0}, Timestamp: now.AddDate(0, 0, 30)}, // 30 days in future, weight ~2.0
		{Vector: []float32{0.0}, Timestamp: now},                   // today, weight 1.0
	}

	result := ComputeTemporallyWeightedVector(vectors, 30, now)
	require.NotNil(t, result)
	// weights: 2.0 and 1.0, total = 3.0
	// result: [1.0*2.0 + 0.0*1.0] / 3.0 = 0.667
	assert.InDelta(t, 0.667, result[0], 0.01)
}
