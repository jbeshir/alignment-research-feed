package controller

import (
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
)

func TestArticleReadSet_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name       string
		articleID  string
		readValue  string
		userID     string
		articles   []domain.Article
		fetchErr   error
		setErr     error
		wantStatus int
		skipFetch  bool
		skipSet    bool
	}{
		{
			name:      "set_read_true",
			articleID: "hash123",
			readValue: "true",
			userID:    "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:      "set_read_false",
			articleID: "hash123",
			readValue: "false",
			userID:    "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid_read_value",
			articleID:  "hash123",
			readValue:  "invalid",
			userID:     "user456",
			wantStatus: http.StatusBadRequest,
			skipFetch:  true,
			skipSet:    true,
		},
		{
			name:       "fetch_error",
			articleID:  "hash123",
			readValue:  "true",
			userID:     "user456",
			fetchErr:   errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
			skipSet:    true,
		},
		{
			name:      "set_error",
			articleID: "hash123",
			readValue: "true",
			userID:    "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			setErr:     errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewMockArticleFetcher(t)
			readSetter := mocks.NewMockArticleReadSetter(t)

			if !tc.skipFetch {
				fetcher.EXPECT().
					FetchArticlesByID(mock.Anything, []string{tc.articleID}).
					Return(tc.articles, tc.fetchErr)
			}

			if !tc.skipSet && tc.fetchErr == nil {
				expectedValue := tc.readValue == boolTrue
				readSetter.EXPECT().
					SetArticleRead(mock.Anything, tc.articleID, tc.userID, expectedValue).
					Return(tc.setErr)
			}

			controller := ArticleReadSet{
				Fetcher:    fetcher,
				ReadSetter: readSetter,
			}

			req := httptest.NewRequest(http.MethodPost, "/articles/"+tc.articleID+"/read/"+tc.readValue, nil)
			req = testContextWithUserID(tc.userID)(req)
			req = mux.SetURLVars(req, map[string]string{
				"article_id": tc.articleID,
				"read":       tc.readValue,
			})
			rec := httptest.NewRecorder()

			controller.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
