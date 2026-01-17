package controller

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testContext() func(r *http.Request) *http.Request {
	return func(r *http.Request) *http.Request {
		ctx := domain.ContextWithLogger(r.Context(), slog.New(slog.DiscardHandler))
		return r.WithContext(ctx)
	}
}

func testContextWithUserID(userID string) func(r *http.Request) *http.Request {
	return func(r *http.Request) *http.Request {
		ctx := domain.ContextWithLogger(r.Context(), slog.New(slog.DiscardHandler))
		ctx = domain.ContextWithUserID(ctx, userID)
		return r.WithContext(ctx)
	}
}

type mockArticlesListLister struct {
	*mocks.MockLatestArticleLister
	*mocks.MockArticleFetcher
}

func TestArticlesList_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name          string
		queryString   string
		setupContext  func(r *http.Request) *http.Request
		articleIDs    []string
		listIDsErr    error
		articles      []domain.Article
		fetchErr      error
		wantStatus    int
		wantCacheCtrl string
		wantArticles  []domain.Article
		skipListIDs   bool
		skipFetch     bool
	}{
		{
			name:         "successful_list",
			queryString:  "",
			setupContext: testContext(),
			articleIDs:   []string{"hash1", "hash2"},
			articles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
				{HashID: "hash2", Title: "Article 2", PublishedAt: testTime},
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "max-age=3600",
			wantArticles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
				{HashID: "hash2", Title: "Article 2", PublishedAt: testTime},
			},
		},
		{
			name:         "no_cache_for_authenticated_user",
			queryString:  "",
			setupContext: testContextWithUserID("user123"),
			articleIDs:   []string{"hash1"},
			articles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "",
			wantArticles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
			},
		},
		{
			name:          "empty_list",
			queryString:   "",
			setupContext:  testContext(),
			articleIDs:    []string{},
			articles:      []domain.Article{},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "max-age=3600",
			wantArticles:  []domain.Article{},
		},
		{
			name:         "with_source_filter",
			queryString:  "filter_sources_allowlist=lesswrong,alignmentforum",
			setupContext: testContext(),
			articleIDs:   []string{"hash1"},
			articles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", Source: "lesswrong", PublishedAt: testTime},
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "max-age=3600",
			wantArticles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", Source: "lesswrong", PublishedAt: testTime},
			},
		},
		{
			name:         "with_pagination",
			queryString:  "page=2&page_size=10",
			setupContext: testContext(),
			articleIDs:   []string{"hash1"},
			articles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
			},
			wantStatus:    http.StatusOK,
			wantCacheCtrl: "max-age=3600",
			wantArticles: []domain.Article{
				{HashID: "hash1", Title: "Article 1", PublishedAt: testTime},
			},
		},
		{
			name:         "invalid_page_param",
			queryString:  "page=invalid",
			setupContext: testContext(),
			wantStatus:   http.StatusBadRequest,
			skipListIDs:  true,
			skipFetch:    true,
		},
		{
			name:         "invalid_page_size_exceeds_limit",
			queryString:  "page_size=500",
			setupContext: testContext(),
			wantStatus:   http.StatusBadRequest,
			skipListIDs:  true,
			skipFetch:    true,
		},
		{
			name:         "invalid_published_after_date",
			queryString:  "filter_published_after=not-a-date",
			setupContext: testContext(),
			wantStatus:   http.StatusBadRequest,
			skipListIDs:  true,
			skipFetch:    true,
		},
		{
			name:         "list_ids_error",
			queryString:  "",
			setupContext: testContext(),
			listIDsErr:   errors.New("database error"),
			wantStatus:   http.StatusInternalServerError,
			skipFetch:    true,
		},
		{
			name:         "fetch_articles_error",
			queryString:  "",
			setupContext: testContext(),
			articleIDs:   []string{"hash1"},
			fetchErr:     errors.New("fetch error"),
			wantStatus:   http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lister := mocks.NewMockLatestArticleLister(t)
			fetcher := mocks.NewMockArticleFetcher(t)

			if !tc.skipListIDs {
				lister.EXPECT().
					ListLatestArticleIDs(mock.Anything, mock.Anything, mock.Anything).
					Return(tc.articleIDs, tc.listIDsErr)
			}

			if !tc.skipFetch && tc.listIDsErr == nil {
				fetcher.EXPECT().
					FetchArticlesByID(mock.Anything, tc.articleIDs).
					Return(tc.articles, tc.fetchErr)
			}

			controller := ArticlesList{
				Lister: &mockArticlesListLister{
					MockLatestArticleLister: lister,
					MockArticleFetcher:      fetcher,
				},
				CacheMaxAge: time.Hour,
			}

			req := httptest.NewRequest(http.MethodGet, "/articles?"+tc.queryString, nil)
			req = tc.setupContext(req)
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
