package command

import (
	"context"
	"errors"
	"testing"

	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDivideVector(t *testing.T) {
	cases := []struct {
		name     string
		vector   []float32
		divisor  float32
		expected []float32
	}{
		{
			name:     "simple_division",
			vector:   []float32{2.0, 4.0, 6.0},
			divisor:  2.0,
			expected: []float32{1.0, 2.0, 3.0},
		},
		{
			name:     "divide_by_one",
			vector:   []float32{1.0, 2.0, 3.0},
			divisor:  1.0,
			expected: []float32{1.0, 2.0, 3.0},
		},
		{
			name:     "empty_vector",
			vector:   []float32{},
			divisor:  2.0,
			expected: []float32{},
		},
		{
			name:     "fractional_result",
			vector:   []float32{1.0, 2.0, 3.0},
			divisor:  2.0,
			expected: []float32{0.5, 1.0, 1.5},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := divideVector(tc.vector, tc.divisor)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRecommendArticles_Execute(t *testing.T) {
	cases := []struct {
		name        string
		vectorSum   []float32
		vectorCount int
		similar     []domain.SimilarArticle
		articles    []domain.Article
		expected    []domain.Article
		wantErr     bool
		errContains string
	}{
		{
			name:        "no_vector_returns_nil",
			vectorSum:   nil,
			vectorCount: 0,
			expected:    nil,
		},
		{
			name:        "zero_count_returns_nil",
			vectorSum:   []float32{1.0, 2.0},
			vectorCount: 0,
			expected:    nil,
		},
		{
			name:        "successful_recommendation",
			vectorSum:   []float32{2.0, 4.0, 6.0},
			vectorCount: 2,
			similar: []domain.SimilarArticle{
				{HashID: "rec1", Score: 0.9},
				{HashID: "rec2", Score: 0.8},
			},
			articles: []domain.Article{
				{HashID: "rec1", Title: "Recommended 1"},
				{HashID: "rec2", Title: "Recommended 2"},
			},
			expected: []domain.Article{
				{HashID: "rec1", Title: "Recommended 1"},
				{HashID: "rec2", Title: "Recommended 2"},
			},
		},
		{
			name:        "no_similar_articles",
			vectorSum:   []float32{1.0, 2.0},
			vectorCount: 1,
			similar:     []domain.SimilarArticle{},
			expected:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vectorSimilarity := mocks.NewMockSimilarArticlesByVectorLister(t)
			articleFetcher := mocks.NewMockArticleFetcher(t)
			userVectorGetter := mocks.NewMockUserVectorGetter(t)

			userVectorGetter.EXPECT().
				GetUserVector(mock.Anything, "user1").
				Return(tc.vectorSum, tc.vectorCount, nil)

			// Only expect similarity and fetch calls when we have a valid vector
			if tc.vectorSum != nil && tc.vectorCount > 0 {
				vectorSimilarity.EXPECT().
					ListSimilarArticlesByVector(mock.Anything, mock.Anything, mock.Anything, 10).
					Return(tc.similar, nil)

				if len(tc.similar) > 0 {
					articleFetcher.EXPECT().
						FetchArticlesByID(mock.Anything, mock.Anything).
						Return(tc.articles, nil)
				}
			}

			cmd := &RecommendArticles{
				VectorSimilarity: vectorSimilarity,
				ArticleFetcher:   articleFetcher,
				UserVectorGetter: userVectorGetter,
			}

			result, err := cmd.Execute(context.Background(), "user1", 10)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestRecommendArticles_Execute_GetUserVectorError(t *testing.T) {
	userVectorGetter := mocks.NewMockUserVectorGetter(t)
	userVectorGetter.EXPECT().
		GetUserVector(mock.Anything, "user1").
		Return(nil, 0, errors.New("db error"))

	cmd := &RecommendArticles{
		UserVectorGetter: userVectorGetter,
	}

	_, err := cmd.Execute(context.Background(), "user1", 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting user vector")
}
