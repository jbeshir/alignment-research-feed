package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSemanticSearch_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)
	testVector := []float32{0.1, 0.2, 0.3}

	cases := []struct {
		name          string
		body          string
		embedVector   []float32
		embedErr      error
		similarResult []domain.SimilarArticle
		similarErr    error
		articles      []domain.Article
		fetchErr      error
		wantStatus    int
		wantArticles  []domain.Article
		skipEmbed     bool
		skipSimilar   bool
		skipFetch     bool
	}{
		{
			name:        "successful_search",
			body:        `{"text": "alignment research", "limit": 5}`,
			embedVector: testVector,
			similarResult: []domain.SimilarArticle{
				{HashID: "hash1", Score: 0.9},
				{HashID: "hash2", Score: 0.8},
			},
			articles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
				{HashID: "hash2", Title: "Article 2", PublishedAt: testTime},
			},
			wantStatus: http.StatusOK,
			wantArticles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
				{HashID: "hash2", Title: "Article 2", PublishedAt: testTime},
			},
		},
		{
			name:        "empty_text",
			body:        `{"text": ""}`,
			wantStatus:  http.StatusBadRequest,
			skipEmbed:   true,
			skipSimilar: true,
			skipFetch:   true,
		},
		{
			name:        "missing_text",
			body:        `{}`,
			wantStatus:  http.StatusBadRequest,
			skipEmbed:   true,
			skipSimilar: true,
			skipFetch:   true,
		},
		{
			name:        "embed_error",
			body:        `{"text": "test"}`,
			embedErr:    errors.New("voyageai error"),
			wantStatus:  http.StatusInternalServerError,
			skipSimilar: true,
			skipFetch:   true,
		},
		{
			name:        "nil_vector_returns_503",
			body:        `{"text": "test"}`,
			embedVector: nil,
			wantStatus:  http.StatusServiceUnavailable,
			skipSimilar: true,
			skipFetch:   true,
		},
		{
			name:        "similarity_error",
			body:        `{"text": "test"}`,
			embedVector: testVector,
			similarErr:  errors.New("pinecone error"),
			wantStatus:  http.StatusInternalServerError,
			skipFetch:   true,
		},
		{
			name:        "fetch_error",
			body:        `{"text": "test"}`,
			embedVector: testVector,
			similarResult: []domain.SimilarArticle{
				{HashID: "hash1", Score: 0.9},
			},
			fetchErr:   errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:        "default_limit",
			body:        `{"text": "test"}`,
			embedVector: testVector,
			similarResult: []domain.SimilarArticle{
				{HashID: "hash1", Score: 0.9},
			},
			articles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
			},
			wantStatus: http.StatusOK,
			wantArticles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			embedder := mocks.NewMockEmbedder(t)
			similarity := mocks.NewMockSimilarArticlesByVectorLister(t)
			fetcher := mocks.NewMockArticleFetcher(t)

			if !tc.skipEmbed {
				embedder.EXPECT().
					EmbedText(mock.Anything, mock.Anything).
					Return(tc.embedVector, tc.embedErr)
			}

			if !tc.skipSimilar {
				similarity.EXPECT().
					ListSimilarArticlesByVector(mock.Anything, []string(nil), tc.embedVector, mock.Anything).
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

			controller := SemanticSearch{
				Embedder:   embedder,
				Similarity: similarity,
				Fetcher:    fetcher,
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/articles/semantic-search", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = testContext()(req)
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
