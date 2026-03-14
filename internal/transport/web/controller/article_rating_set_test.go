package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	cmdmocks "github.com/jbeshir/alignment-research-feed/internal/command/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestArticleRatingSet_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name          string
		ratingType    domain.UserRatingType
		articleID     string
		ratingValue   string
		userID        string
		articles      []domain.Article
		fetchErr      error
		setRatingErr  error
		wantStatus    int
		skipFetch     bool
		skipSetRating bool
	}{
		{
			name:        "thumbs_up_true",
			ratingType:  domain.RatingTypeThumbsUp,
			articleID:   "hash123",
			ratingValue: "true",
			userID:      "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: &testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:        "thumbs_up_false",
			ratingType:  domain.RatingTypeThumbsUp,
			articleID:   "hash123",
			ratingValue: "false",
			userID:      "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: &testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:          "thumbs_up_invalid",
			ratingType:    domain.RatingTypeThumbsUp,
			articleID:     "hash123",
			ratingValue:   "invalid",
			userID:        "user456",
			wantStatus:    http.StatusBadRequest,
			skipFetch:     true,
			skipSetRating: true,
		},
		{
			name:        "thumbs_down_true",
			ratingType:  domain.RatingTypeThumbsDown,
			articleID:   "hash123",
			ratingValue: "true",
			userID:      "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: &testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:        "thumbs_down_false",
			ratingType:  domain.RatingTypeThumbsDown,
			articleID:   "hash123",
			ratingValue: "false",
			userID:      "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: &testTime},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:          "thumbs_down_invalid",
			ratingType:    domain.RatingTypeThumbsDown,
			articleID:     "hash123",
			ratingValue:   "invalid",
			userID:        "user456",
			wantStatus:    http.StatusBadRequest,
			skipFetch:     true,
			skipSetRating: true,
		},
		{
			name:          "fetch_error",
			ratingType:    domain.RatingTypeThumbsUp,
			articleID:     "hash123",
			ratingValue:   "true",
			userID:        "user456",
			fetchErr:      errors.New("database error"),
			wantStatus:    http.StatusInternalServerError,
			skipSetRating: true,
		},
		{
			name:        "set_rating_error",
			ratingType:  domain.RatingTypeThumbsUp,
			articleID:   "hash123",
			ratingValue: "true",
			userID:      "user456",
			articles: []domain.Article{
				{HashID: "hash123", Title: "Test", PublishedAt: &testTime},
			},
			setRatingErr: errors.New("database error"),
			wantStatus:   http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewArticleFetcher(t)
			setRatingCmd := cmdmocks.NewCommand[command.SetArticleRatingRequest, command.Empty](t)

			if !tc.skipFetch {
				fetcher.EXPECT().
					FetchArticlesByID(mock.Anything, []string{tc.articleID}).
					Return(tc.articles, tc.fetchErr)
			}

			if !tc.skipSetRating && tc.fetchErr == nil {
				ratingEnabled := tc.ratingValue == boolTrue
				req := command.SetArticleRatingRequest{
					UserID:        tc.userID,
					ArticleHashID: tc.articleID,
				}
				switch tc.ratingType {
				case domain.RatingTypeThumbsUp:
					req.ThumbsUp = ratingEnabled
				default:
					req.ThumbsDown = ratingEnabled
				}
				setRatingCmd.EXPECT().
					Execute(mock.Anything, req).
					Return(command.Empty{}, tc.setRatingErr)
			}

			ctrl := ArticleRatingSet{
				Fetcher:      fetcher,
				SetRatingCmd: setRatingCmd,
				RatingType:   tc.ratingType,
			}

			paramName := string(tc.ratingType)
			urlPath := "/articles/" + tc.articleID + "/" + paramName + "/" + tc.ratingValue

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, urlPath, nil)
			req = testContextWithUserID(tc.userID)(req)
			req = mux.SetURLVars(req, map[string]string{
				"article_id": tc.articleID,
				paramName:    tc.ratingValue,
			})
			rec := httptest.NewRecorder()

			ctrl.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
