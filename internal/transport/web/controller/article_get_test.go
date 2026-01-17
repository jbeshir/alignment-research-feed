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

func TestArticleGet_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name          string
		articleID     string
		setupContext  func(r *http.Request) *http.Request
		articles      []domain.Article
		fetchErr      error
		wantStatus    int
		wantCacheCtrl string
		wantArticle   *domain.Article
	}{
		{
			name:         "successful_fetch",
			articleID:    "hash123",
			setupContext: testContext(),
			articles: []domain.Article{
				{
					HashID:      "hash123",
					Title:       "Test Article",
					Link:        "https://example.com/article",
					Source:      "lesswrong",
					Authors:     "John Doe",
					PublishedAt: testTime,
				},
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "max-age=3600",
			wantArticle: &domain.Article{
				HashID:      "hash123",
				Title:       "Test Article",
				Link:        "https://example.com/article",
				Source:      "lesswrong",
				Authors:     "John Doe",
				PublishedAt: testTime,
			},
		},
		{
			name:         "no_cache_for_authenticated_user",
			articleID:    "hash123",
			setupContext: testContextWithUserID("user456"),
			articles: []domain.Article{
				{
					HashID:      "hash123",
					Title:       "Test Article",
					PublishedAt: testTime,
				},
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "",
			wantArticle: &domain.Article{
				HashID:      "hash123",
				Title:       "Test Article",
				PublishedAt: testTime,
			},
		},
		{
			name:         "fetch_error",
			articleID:    "hash123",
			setupContext: testContext(),
			fetchErr:     errors.New("database error"),
			wantStatus:   http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewMockArticleFetcher(t)

			fetcher.EXPECT().
				FetchArticlesByID(mock.Anything, []string{tc.articleID}).
				Return(tc.articles, tc.fetchErr)

			controller := ArticleGet{
				Fetcher:     fetcher,
				CacheMaxAge: time.Hour,
			}

			req := httptest.NewRequest(http.MethodGet, "/articles/"+tc.articleID, nil)
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

				var article domain.Article
				err := json.NewDecoder(rec.Body).Decode(&article)
				require.NoError(t, err)
				assert.Equal(t, *tc.wantArticle, article)
			}
		})
	}
}
