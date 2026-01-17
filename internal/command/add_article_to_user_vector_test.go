package command

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestAddArticleToUserVector_Execute(t *testing.T) {
	testVector := []float32{0.1, 0.2, 0.3}

	cases := []struct {
		name          string
		userID        string
		articleHashID string
		vector        []float32
		vectorErr     error
		addReturns    bool
		addErr        error
		wantAddCall   bool
	}{
		{
			name:          "adds_vector",
			userID:        "user1",
			articleHashID: "article1",
			vector:        testVector,
			addReturns:    true,
			wantAddCall:   true,
		},
		{
			name:          "already_added_no_change",
			userID:        "user1",
			articleHashID: "article1",
			vector:        testVector,
			addReturns:    false, // syncer returns false when already added
			wantAddCall:   true,
		},
		{
			name:          "no_vector_skips",
			userID:        "user1",
			articleHashID: "article1",
			vector:        nil,
			wantAddCall:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewMockArticleVectorFetcher(t)
			syncer := mocks.NewMockUserVectorSyncer(t)

			fetcher.EXPECT().
				FetchArticleVector(mock.Anything, tc.articleHashID).
				Return(tc.vector, tc.vectorErr)

			if tc.wantAddCall {
				syncer.EXPECT().
					AddArticleVectorToUser(mock.Anything, tc.userID, tc.articleHashID, tc.vector).
					Return(tc.addReturns, tc.addErr)
			}

			cmd := &AddArticleToUserVector{
				ArticleVectorFetcher: fetcher,
				UserVectorSyncer:     syncer,
			}

			ctx := domain.ContextWithLogger(context.Background(), testLogger())
			err := cmd.Execute(ctx, tc.userID, tc.articleHashID)
			require.NoError(t, err)
		})
	}
}

func TestAddArticleToUserVector_Execute_FetchError(t *testing.T) {
	fetcher := mocks.NewMockArticleVectorFetcher(t)
	syncer := mocks.NewMockUserVectorSyncer(t)

	fetcher.EXPECT().
		FetchArticleVector(mock.Anything, "article1").
		Return(nil, errors.New("pinecone error"))

	cmd := &AddArticleToUserVector{
		ArticleVectorFetcher: fetcher,
		UserVectorSyncer:     syncer,
	}

	ctx := domain.ContextWithLogger(context.Background(), testLogger())
	err := cmd.Execute(ctx, "user1", "article1")

	// Fetch errors are logged but not returned
	require.NoError(t, err)
}

func TestAddArticleToUserVector_Execute_AddError(t *testing.T) {
	fetcher := mocks.NewMockArticleVectorFetcher(t)
	syncer := mocks.NewMockUserVectorSyncer(t)

	testVector := []float32{0.1}

	fetcher.EXPECT().
		FetchArticleVector(mock.Anything, "article1").
		Return(testVector, nil)

	syncer.EXPECT().
		AddArticleVectorToUser(mock.Anything, "user1", "article1", testVector).
		Return(false, errors.New("db error"))

	cmd := &AddArticleToUserVector{
		ArticleVectorFetcher: fetcher,
		UserVectorSyncer:     syncer,
	}

	ctx := domain.ContextWithLogger(context.Background(), testLogger())
	err := cmd.Execute(ctx, "user1", "article1")

	require.Error(t, err)
	require.Contains(t, err.Error(), "adding article vector to user")
}
