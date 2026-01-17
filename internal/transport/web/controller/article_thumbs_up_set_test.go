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

func TestArticleThumbsUpSet_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name            string
		articleID       string
		thumbsUpValue   string
		userID          string
		articles        []domain.Article
		fetchErr        error
		setThumbsUpErr  error
		addVectorErr    error
		removeVectorErr error
		wantStatus      int
		skipFetch       bool
		skipSetThumbsUp bool
		skipAddVector   bool
		skipRemoveVec   bool
	}{
		{
			name:          "set_thumbs_up_true",
			articleID:     "hash123",
			thumbsUpValue: "true",
			userID:        "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			wantStatus:    http.StatusNoContent,
			skipRemoveVec: true,
		},
		{
			name:          "set_thumbs_up_false",
			articleID:     "hash123",
			thumbsUpValue: "false",
			userID:        "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			wantStatus:    http.StatusNoContent,
			skipAddVector: true,
		},
		{
			name:            "invalid_thumbs_up_value",
			articleID:       "hash123",
			thumbsUpValue:   "invalid",
			userID:          "user456",
			wantStatus:      http.StatusBadRequest,
			skipFetch:       true,
			skipSetThumbsUp: true,
			skipAddVector:   true,
			skipRemoveVec:   true,
		},
		{
			name:            "fetch_error",
			articleID:       "hash123",
			thumbsUpValue:   "true",
			userID:          "user456",
			fetchErr:        errors.New("database error"),
			wantStatus:      http.StatusInternalServerError,
			skipSetThumbsUp: true,
			skipAddVector:   true,
			skipRemoveVec:   true,
		},
		{
			name:          "set_thumbs_up_error",
			articleID:     "hash123",
			thumbsUpValue: "true",
			userID:        "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			setThumbsUpErr: errors.New("database error"),
			wantStatus:     http.StatusInternalServerError,
			skipAddVector:  true,
			skipRemoveVec:  true,
		},
		{
			name:          "add_vector_error",
			articleID:     "hash123",
			thumbsUpValue: "true",
			userID:        "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			addVectorErr:  errors.New("vector error"),
			wantStatus:    http.StatusInternalServerError,
			skipRemoveVec: true,
		},
		{
			name:          "remove_vector_error",
			articleID:     "hash123",
			thumbsUpValue: "false",
			userID:        "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: testTime},
			},
			removeVectorErr: errors.New("vector error"),
			wantStatus:      http.StatusInternalServerError,
			skipAddVector:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewMockArticleFetcher(t)
			thumbsUpSetter := mocks.NewMockArticleThumbsUpSetter(t)
			vectorFetcher := mocks.NewMockArticleVectorFetcher(t)
			vectorSyncer := mocks.NewMockUserVectorSyncer(t)

			if !tc.skipFetch {
				fetcher.EXPECT().
					FetchArticlesByID(mock.Anything, []string{tc.articleID}).
					Return(tc.articles, tc.fetchErr)
			}

			if !tc.skipSetThumbsUp && tc.fetchErr == nil {
				expectedValue := tc.thumbsUpValue == boolTrue
				thumbsUpSetter.EXPECT().
					SetArticleThumbsUp(mock.Anything, tc.articleID, tc.userID, expectedValue).
					Return(tc.setThumbsUpErr)
			}

			// Set up vector command mocks
			if !tc.skipAddVector && tc.setThumbsUpErr == nil {
				vectorFetcher.EXPECT().
					FetchArticleVector(mock.Anything, tc.articleID).
					Return([]float32{0.1, 0.2}, nil)
				if tc.addVectorErr != nil {
					vectorSyncer.EXPECT().
						AddArticleVectorToUser(mock.Anything, tc.userID, tc.articleID, mock.Anything).
						Return(false, tc.addVectorErr)
				} else {
					vectorSyncer.EXPECT().
						AddArticleVectorToUser(mock.Anything, tc.userID, tc.articleID, mock.Anything).
						Return(true, nil)
				}
			}

			if !tc.skipRemoveVec && tc.setThumbsUpErr == nil {
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

			controller := ArticleThumbsUpSet{
				Fetcher:        fetcher,
				ThumbsUpSetter: thumbsUpSetter,
				AddVectorCmd: &command.AddArticleToUserVector{
					ArticleVectorFetcher: vectorFetcher,
					UserVectorSyncer:     vectorSyncer,
				},
				RemoveVectorCmd: &command.RemoveArticleFromUserVector{
					ArticleVectorFetcher: vectorFetcher,
					UserVectorSyncer:     vectorSyncer,
				},
			}

			req := httptest.NewRequest(http.MethodPost, "/articles/"+tc.articleID+"/thumbs-up/"+tc.thumbsUpValue, nil)
			req = testContextWithUserID(tc.userID)(req)
			req = mux.SetURLVars(req, map[string]string{
				"article_id": tc.articleID,
				"thumbs_up":  tc.thumbsUpValue,
			})
			rec := httptest.NewRecorder()

			controller.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
