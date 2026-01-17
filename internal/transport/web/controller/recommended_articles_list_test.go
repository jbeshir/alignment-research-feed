package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRecommendedArticlesList_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name          string
		userID        string
		vectorSum     []float32
		vectorCount   int
		vectorErr     error
		similarResult []domain.SimilarArticle
		similarErr    error
		articles      []domain.Article
		fetchErr      error
		wantStatus    int
		wantArticles  []domain.Article
		skipVector    bool
		skipSimilar   bool
		skipFetch     bool
	}{
		{
			name:        "successful_recommendations",
			userID:      "user456",
			vectorSum:   []float32{0.2, 0.4, 0.6},
			vectorCount: 2,
			similarResult: []domain.SimilarArticle{
				{HashID: "rec1", Score: 0.9},
				{HashID: "rec2", Score: 0.8},
			},
			articles: []domain.Article{
				{HashID: "rec1", Title: "Recommended 1", PublishedAt: testTime},
				{HashID: "rec2", Title: "Recommended 2", PublishedAt: testTime},
			},
			wantStatus: http.StatusOK,
			wantArticles: []domain.Article{
				{HashID: "rec1", Title: "Recommended 1", PublishedAt: testTime},
				{HashID: "rec2", Title: "Recommended 2", PublishedAt: testTime},
			},
		},
		{
			name:        "no_user_id_unauthorized",
			userID:      "",
			wantStatus:  http.StatusUnauthorized,
			skipVector:  true,
			skipSimilar: true,
			skipFetch:   true,
		},
		{
			name:         "no_vector_empty_recommendations",
			userID:       "user456",
			vectorSum:    nil,
			vectorCount:  0,
			wantStatus:   http.StatusOK,
			wantArticles: []domain.Article{},
			skipSimilar:  true,
			skipFetch:    true,
		},
		{
			name:         "zero_count_empty_recommendations",
			userID:       "user456",
			vectorSum:    []float32{0.1, 0.2},
			vectorCount:  0,
			wantStatus:   http.StatusOK,
			wantArticles: []domain.Article{},
			skipSimilar:  true,
			skipFetch:    true,
		},
		{
			name:        "vector_error",
			userID:      "user456",
			vectorErr:   errors.New("database error"),
			wantStatus:  http.StatusInternalServerError,
			skipSimilar: true,
			skipFetch:   true,
		},
		{
			name:          "no_similar_articles",
			userID:        "user456",
			vectorSum:     []float32{0.1, 0.2},
			vectorCount:   1,
			similarResult: []domain.SimilarArticle{},
			wantStatus:    http.StatusOK,
			wantArticles:  []domain.Article{},
			skipFetch:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vectorGetter := mocks.NewMockUserVectorGetter(t)
			vectorSimilarity := mocks.NewMockSimilarArticlesByVectorLister(t)
			articleFetcher := mocks.NewMockArticleFetcher(t)

			if !tc.skipVector {
				vectorGetter.EXPECT().
					GetUserVector(mock.Anything, tc.userID).
					Return(tc.vectorSum, tc.vectorCount, tc.vectorErr)
			}

			if !tc.skipSimilar && tc.vectorErr == nil && tc.vectorSum != nil && tc.vectorCount > 0 {
				vectorSimilarity.EXPECT().
					ListSimilarArticlesByVector(mock.Anything, mock.Anything, mock.Anything, 10).
					Return(tc.similarResult, tc.similarErr)
			}

			if !tc.skipFetch && len(tc.similarResult) > 0 {
				var hashIDs []string
				for _, s := range tc.similarResult {
					hashIDs = append(hashIDs, s.HashID)
				}
				articleFetcher.EXPECT().
					FetchArticlesByID(mock.Anything, hashIDs).
					Return(tc.articles, tc.fetchErr)
			}

			controller := RecommendedArticlesList{
				Command: &command.RecommendArticles{
					VectorSimilarity: vectorSimilarity,
					ArticleFetcher:   articleFetcher,
					UserVectorGetter: vectorGetter,
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/recommended", nil)
			if tc.userID != "" {
				req = testContextWithUserID(tc.userID)(req)
			} else {
				req = testContext()(req)
			}
			rec := httptest.NewRecorder()

			controller.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantStatus == http.StatusOK {
				var response ArticlesListResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tc.wantArticles, response.Data)
			}
		})
	}
}
