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
			articleID: testArticleID,
			readValue: boolTrue,
			userID:    testUserID,
			articles: []domain.Article{
				{HashID: testArticleID, Title: testTitle, PublishedAt: &testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:      "set_read_false",
			articleID: testArticleID,
			readValue: boolFalse,
			userID:    testUserID,
			articles: []domain.Article{
				{HashID: testArticleID, Title: testTitle, PublishedAt: &testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid_read_value",
			articleID:  testArticleID,
			readValue:  testInvalidValue,
			userID:     testUserID,
			wantStatus: http.StatusBadRequest,
			skipFetch:  true,
			skipSet:    true,
		},
		{
			name:       testFetchErrorCaseName,
			articleID:  testArticleID,
			readValue:  boolTrue,
			userID:     testUserID,
			fetchErr:   errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
			skipSet:    true,
		},
		{
			name:      "set_error",
			articleID: testArticleID,
			readValue: boolTrue,
			userID:    testUserID,
			articles: []domain.Article{
				{HashID: testArticleID, Title: testTitle, PublishedAt: &testTime},
			},
			setErr:     errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewArticleFetcher(t)
			readSetter := mocks.NewArticleReadSetter(t)

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

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/articles/"+tc.articleID+"/read/"+tc.readValue, nil)
			req = testContextWithUserID(tc.userID)(req)
			req = mux.SetURLVars(req, map[string]string{
				muxVarArticleID: tc.articleID,
				"read":          tc.readValue,
			})
			rec := httptest.NewRecorder()

			controller.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
