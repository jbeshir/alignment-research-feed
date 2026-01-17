package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestArticleThumbsDownSet_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name              string
		articleID         string
		thumbsDownValue   string
		userID            string
		articles          []domain.Article
		fetchErr          error
		setThumbsDownErr  error
		removeVectorErr   error
		wantStatus        int
		skipFetch         bool
		skipSetThumbsDown bool
		skipRemoveVec     bool
	}{
		{
			name:            "set_thumbs_down_true",
			articleID:       "hash123",
			thumbsDownValue: "true",
			userID:          "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:            "set_thumbs_down_false_no_vector_removal",
			articleID:       "hash123",
			thumbsDownValue: "false",
			userID:          "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			wantStatus:    http.StatusNoContent,
			skipRemoveVec: true,
		},
		{
			name:              "invalid_thumbs_down_value",
			articleID:         "hash123",
			thumbsDownValue:   "invalid",
			userID:            "user456",
			wantStatus:        http.StatusBadRequest,
			skipFetch:         true,
			skipSetThumbsDown: true,
			skipRemoveVec:     true,
		},
		{
			name:              "fetch_error",
			articleID:         "hash123",
			thumbsDownValue:   "true",
			userID:            "user456",
			fetchErr:          errors.New("database error"),
			wantStatus:        http.StatusInternalServerError,
			skipSetThumbsDown: true,
			skipRemoveVec:     true,
		},
		{
			name:            "set_thumbs_down_error",
			articleID:       "hash123",
			thumbsDownValue: "true",
			userID:          "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			setThumbsDownErr: errors.New("database error"),
			wantStatus:       http.StatusInternalServerError,
			skipRemoveVec:    true,
		},
		{
			name:            "remove_vector_error",
			articleID:       "hash123",
			thumbsDownValue: "true",
			userID:          "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			removeVectorErr: errors.New("vector error"),
			wantStatus:      http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewMockArticleFetcher(t)
			thumbsDownSetter := mocks.NewMockArticleThumbsDownSetter(t)
			vectorFetcher := mocks.NewMockArticleVectorFetcher(t)
			vectorSyncer := mocks.NewMockUserVectorSyncer(t)

			if !tc.skipFetch {
				fetcher.EXPECT().
					FetchArticlesByID(mock.Anything, []string{tc.articleID}).
					Return(tc.articles, tc.fetchErr)
			}

			if !tc.skipSetThumbsDown && tc.fetchErr == nil {
				expectedValue := tc.thumbsDownValue == boolTrue
				thumbsDownSetter.EXPECT().
					SetArticleThumbsDown(mock.Anything, tc.articleID, tc.userID, expectedValue).
					Return(tc.setThumbsDownErr)
			}

			// Only remove vector when thumbs_down is set to true
			if !tc.skipRemoveVec && tc.setThumbsDownErr == nil {
				vectorFetcher.EXPECT().
					FetchArticleVector(mock.Anything, tc.articleID).
					Return([]float32{0.1, 0.2}, nil)
				if tc.removeVectorErr != nil {
					vectorSyncer.EXPECT().
						SubtractArticleVectorFromUser(mock.Anything, tc.userID, tc.articleID, mock.Anything).
						Return(false, tc.removeVectorErr)
				} else {
					vectorSyncer.EXPECT().
						SubtractArticleVectorFromUser(mock.Anything, tc.userID, tc.articleID, mock.Anything).
						Return(true, nil)
				}
			}

			controller := ArticleThumbsDownSet{
				Fetcher:          fetcher,
				ThumbsDownSetter: thumbsDownSetter,
				RemoveVectorCmd: &command.RemoveArticleFromUserVector{
					ArticleVectorFetcher: vectorFetcher,
					UserVectorSyncer:     vectorSyncer,
				},
			}

			req := httptest.NewRequest(http.MethodPost, "/articles/"+tc.articleID+"/thumbs-down/"+tc.thumbsDownValue, nil)
			req = testContextWithUserID(tc.userID)(req)
			req = mux.SetURLVars(req, map[string]string{
				"article_id":  tc.articleID,
				"thumbs_down": tc.thumbsDownValue,
			})
			rec := httptest.NewRecorder()

			controller.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
