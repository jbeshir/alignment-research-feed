package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testGenerateRecommendationsConfig returns a config for testing.
func testGenerateRecommendationsConfig() GenerateRecommendationsConfig {
	return GenerateRecommendationsConfig{
		TemporalDecayHalfLifeDays: 90,
		NegativeSignalWeight:      0.3,
		UseInterestClusters:       true,
		CandidatesPerCluster:      20,
	}
}

// assertScoredArticlesEqual compares ScoredArticle slices ignoring Source field differences.
func assertScoredArticlesEqual(t *testing.T, expected, actual []ScoredArticle) {
	t.Helper()
	if expected == nil && actual == nil {
		return
	}
	require.Len(t, actual, len(expected), "length mismatch")
	for i := range expected {
		assert.Equal(t, expected[i].HashID, actual[i].HashID, "HashID mismatch at index %d", i)
		assert.InDelta(t, expected[i].Score, actual[i].Score, 0.01, "Score mismatch at index %d", i)
	}
}

func TestGenerateRecommendations_Execute(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name           string
		thumbsUpVecs   []domain.UserArticleRating
		thumbsUpErr    error
		similar        []domain.SimilarArticle
		expected       []ScoredArticle
		wantErr        bool
		errContains    string
		skipSimilarity bool
	}{
		{
			name:           "no_thumbs_up_returns_nil",
			thumbsUpVecs:   nil,
			expected:       nil,
			skipSimilarity: true,
		},
		{
			name:           "empty_thumbs_up_returns_nil",
			thumbsUpVecs:   []domain.UserArticleRating{},
			expected:       nil,
			skipSimilarity: true,
		},
		{
			name: "successful_recommendation",
			thumbsUpVecs: []domain.UserArticleRating{
				{ArticleHashID: "art1", Vector: []float32{1.0, 0.0, 0.0}, RatedAt: now},
				{ArticleHashID: "art2", Vector: []float32{0.0, 1.0, 0.0}, RatedAt: now},
			},
			similar: []domain.SimilarArticle{
				{HashID: "rec1", Score: 0.9},
				{HashID: "rec2", Score: 0.8},
			},
			expected: []ScoredArticle{
				{HashID: "rec1", Score: 0.9, Source: "temporal"},
				{HashID: "rec2", Score: 0.8, Source: "temporal"},
			},
		},
		{
			name: "no_similar_articles_returns_nil",
			thumbsUpVecs: []domain.UserArticleRating{
				{ArticleHashID: "art1", Vector: []float32{1.0, 0.0, 0.0}, RatedAt: now},
			},
			similar:  []domain.SimilarArticle{},
			expected: nil,
		},
		{
			name:           "thumbs_up_error",
			thumbsUpErr:    errors.New("database error"),
			wantErr:        true,
			errContains:    "getting thumbs up vectors",
			skipSimilarity: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vectorSimilarity := mocks.NewMockSimilarArticlesByVectorLister(t)
			interactionStore := mocks.NewMockUserArticleInteractionStore(t)
			clusterStore := mocks.NewMockUserInterestClusterStore(t)
			readArticlesLister := mocks.NewMockReadArticleIDsLister(t)

			// Read articles lister is always called first
			readArticlesLister.EXPECT().
				ListReadArticleIDs(mock.Anything, "user1").
				Return([]string{}, nil)

			interactionStore.EXPECT().
				GetUserArticleVectorsByType(mock.Anything, "user1", domain.RatingTypeThumbsUp).
				Return(tc.thumbsUpVecs, tc.thumbsUpErr)

			// When we have thumbs up vectors, also expect thumbs down query and cluster check
			if !tc.skipSimilarity && len(tc.thumbsUpVecs) > 0 {
				interactionStore.EXPECT().
					GetUserArticleVectorsByType(mock.Anything, "user1", domain.RatingTypeThumbsDown).
					Return(nil, nil)

				clusterStore.EXPECT().
					GetUserInterestClusters(mock.Anything, "user1").
					Return(nil, nil)

				vectorSimilarity.EXPECT().
					ListSimilarArticlesByVector(mock.Anything, mock.Anything, mock.Anything, 40).
					Return(tc.similar, nil)
			}

			cmd := NewGenerateRecommendations(
				vectorSimilarity,
				interactionStore,
				clusterStore,
				readArticlesLister,
				testGenerateRecommendationsConfig(),
			)

			result, err := cmd.Execute(context.Background(), GenerateRecommendationsRequest{UserID: "user1", Limit: 10})

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assertScoredArticlesEqual(t, tc.expected, result)
			}
		})
	}
}

func TestGenerateRecommendations_Execute_WithNegativeSignals(t *testing.T) {
	now := time.Now()

	vectorSimilarity := mocks.NewMockSimilarArticlesByVectorLister(t)
	interactionStore := mocks.NewMockUserArticleInteractionStore(t)
	clusterStore := mocks.NewMockUserInterestClusterStore(t)
	readArticlesLister := mocks.NewMockReadArticleIDsLister(t)

	thumbsUpVecs := []domain.UserArticleRating{
		{ArticleHashID: "art1", Vector: []float32{1.0, 0.0, 0.0}, RatedAt: now},
	}
	thumbsDownVecs := []domain.UserArticleRating{
		{ArticleHashID: "bad1", Vector: []float32{0.0, 0.0, 1.0}, RatedAt: now},
	}

	readArticlesLister.EXPECT().
		ListReadArticleIDs(mock.Anything, "user1").
		Return([]string{}, nil)

	interactionStore.EXPECT().
		GetUserArticleVectorsByType(mock.Anything, "user1", domain.RatingTypeThumbsUp).
		Return(thumbsUpVecs, nil)

	interactionStore.EXPECT().
		GetUserArticleVectorsByType(mock.Anything, "user1", domain.RatingTypeThumbsDown).
		Return(thumbsDownVecs, nil)

	clusterStore.EXPECT().
		GetUserInterestClusters(mock.Anything, "user1").
		Return(nil, nil)

	similar := []domain.SimilarArticle{
		{HashID: "rec1", Score: 0.9},
	}

	vectorSimilarity.EXPECT().
		ListSimilarArticlesByVector(mock.Anything, mock.Anything, mock.Anything, 40).
		Return(similar, nil)

	cmd := NewGenerateRecommendations(
		vectorSimilarity,
		interactionStore,
		clusterStore,
		readArticlesLister,
		testGenerateRecommendationsConfig(),
	)

	result, err := cmd.Execute(context.Background(), GenerateRecommendationsRequest{UserID: "user1", Limit: 10})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "rec1", result[0].HashID)
	// Score should be penalized: 0.9 - (0.3 * 0.9 * 0.5) = 0.765
	assert.InDelta(t, 0.765, result[0].Score, 0.01)
}
