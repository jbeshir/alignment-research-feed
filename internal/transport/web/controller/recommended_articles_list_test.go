package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/command"
	cmdmocks "github.com/jbeshir/alignment-research-feed/internal/command/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRecommendedArticlesList_ServeHTTP(t *testing.T) {
	testTime := time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name         string
		userID       string
		articles     []domain.Article
		commandErr   error
		wantStatus   int
		wantArticles []domain.Article
	}{
		{
			name:   "successful_recommendations",
			userID: "user456",
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
			name:       "no_user_id_unauthorized",
			userID:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:         "no_recommendations",
			userID:       "user456",
			articles:     nil,
			wantStatus:   http.StatusOK,
			wantArticles: []domain.Article{},
		},
		{
			name:       "command_error",
			userID:     "user456",
			commandErr: errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recommendCmd := cmdmocks.NewMockCommand[command.RecommendArticlesRequest, []domain.Article](t)

			if tc.userID != "" {
				expectedReq := command.RecommendArticlesRequest{UserID: tc.userID, Limit: recommendationsLimit}
				recommendCmd.EXPECT().
					Execute(mock.Anything, expectedReq).
					Return(tc.articles, tc.commandErr)
			}

			controller := RecommendedArticlesList{
				Command: recommendCmd,
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
