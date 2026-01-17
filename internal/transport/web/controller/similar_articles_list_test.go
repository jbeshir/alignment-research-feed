package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSimilarArticlesList_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name          string
		articleID     string
		setupContext  func(r *http.Request) *http.Request
		similarResult []domain.SimilarArticle
		similarErr    error
		articles      []domain.Article
		fetchErr      error
		wantStatus    int
		wantCacheCtrl string
		wantArticles  []domain.Article
		skipSimilar   bool
		skipFetch     bool
	}{
		{
			name:         "successful_similar_articles",
			articleID:    "hash123",
			setupContext: testContext(),
			similarResult: []domain.SimilarArticle{
				{HashID: "similar1", Score: 0.9},
				{HashID: "similar2", Score: 0.8},
			},
			articles: []domain.Article{
				{HashID: "similar1", Title: "Similar Article 1", PublishedAt: testTime},
				{HashID: "similar2", Title: "Similar Article 2", PublishedAt: testTime},
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "max-age=3600",
			wantArticles: []domain.Article{
				{HashID: "similar1", Title: "Similar Article 1", PublishedAt: testTime},
				{HashID: "similar2", Title: "Similar Article 2", PublishedAt: testTime},
			},
		},
		{
			name:         "no_cache_for_authenticated_user",
			articleID:    "hash123",
			setupContext: testContextWithUserID("user456"),
			similarResult: []domain.SimilarArticle{
				{HashID: "similar1", Score: 0.9},
			},
			articles: []domain.Article{
				{HashID: "similar1", Title: "Similar Article 1", PublishedAt: testTime},
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "",
			wantArticles: []domain.Article{
				{HashID: "similar1", Title: "Similar Article 1", PublishedAt: testTime},
			},
		},
		{
			name:          "empty_similar_articles",
			articleID:     "hash123",
			setupContext:  testContext(),
			similarResult: []domain.SimilarArticle{},
			articles:      []domain.Article{},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "max-age=3600",
			wantArticles:  []domain.Article{},
		},
		{
			name:         "similarity_error",
			articleID:    "hash123",
			setupContext: testContext(),
			similarErr:   errors.New("pinecone error"),
			wantStatus:   http.StatusInternalServerError,
			skipFetch:    true,
		},
		{
			name:         "fetch_error",
			articleID:    "hash123",
			setupContext: testContext(),
			similarResult: []domain.SimilarArticle{
				{HashID: "similar1", Score: 0.9},
			},
			fetchErr:   errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:         "empty_article_id",
			articleID:    "",
			setupContext: testContext(),
			wantStatus:   http.StatusInternalServerError,
			skipSimilar:  true,
			skipFetch:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewMockArticleFetcher(t)
			similarity := mocks.NewMockSimilarArticleLister(t)

			if !tc.skipSimilar {
				similarity.EXPECT().
					ListSimilarArticles(mock.Anything, []string{tc.articleID}, 10).
					Return(tc.similarResult, tc.similarErr)
			}

			if !tc.skipFetch && tc.similarErr == nil {
				hashIDs := make([]string, 0, len(tc.similarResult))
				for _, s := range tc.similarResult {
					hashIDs = append(hashIDs, s.HashID)
				}
				fetcher.EXPECT().
					FetchArticlesByID(mock.Anything, hashIDs).
					Return(tc.articles, tc.fetchErr)
			}

			controller := SimilarArticlesList{
				Fetcher:     fetcher,
				Similarity:  similarity,
				CacheMaxAge: time.Hour,
			}

			req := httptest.NewRequest(http.MethodGet, "/articles/"+tc.articleID+"/similar", nil)
			req = tc.setupContext(req)
			req = mux.SetURLVars(req, map[string]string{"article_id": tc.articleID})
			rec := httptest.NewRecorder()

			controller.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantStatus == http.StatusOK {
				if tc.wantCacheCtrl != "" {
					assert.Equal(t, tc.wantCacheCtrl, rec.Header().Get("Cache-Control"))
				} else {
					assert.Empty(t, rec.Header().Get("Cache-Control"))
				}

				var response ArticlesListResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tc.wantArticles, response.Data)
			}
		})
	}
}
