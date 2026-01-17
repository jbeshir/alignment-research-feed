package command

import (
	"context"
	"errors"
	"testing"

	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRemoveArticleFromUserVector_Execute(t *testing.T) {
	testVector := []float32{0.1, 0.2, 0.3}

	cases := []struct {
		name             string
		userID           string
		articleHashID    string
		vector           []float32
		vectorErr        error
		subtractReturns  bool
		subtractErr      error
		wantSubtractCall bool
	}{
		{
			name:             "subtracts_vector",
			userID:           "user1",
			articleHashID:    "article1",
			vector:           testVector,
			subtractReturns:  true,
			wantSubtractCall: true,
		},
		{
			name:             "not_added_no_change",
			userID:           "user1",
			articleHashID:    "article1",
			vector:           testVector,
			subtractReturns:  false, // syncer returns false when wasn't added
			wantSubtractCall: true,
		},
		{
			name:             "no_vector_clears_flag",
			userID:           "user1",
			articleHashID:    "article1",
			vector:           nil,
			subtractReturns:  true,
			wantSubtractCall: true, // Call is made with nil vector
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := mocks.NewMockArticleVectorFetcher(t)
			syncer := mocks.NewMockUserVectorSyncer(t)

			fetcher.EXPECT().
				FetchArticleVector(mock.Anything, tc.articleHashID).
				Return(tc.vector, tc.vectorErr)

			if tc.wantSubtractCall {
				syncer.EXPECT().
					SubtractArticleVectorFromUser(
						mock.Anything, tc.userID, tc.articleHashID, tc.vector,
					).
					Return(tc.subtractReturns, tc.subtractErr)
			}

			cmd := &RemoveArticleFromUserVector{
				ArticleVectorFetcher: fetcher,
				UserVectorSyncer:     syncer,
			}

			ctx := domain.ContextWithLogger(context.Background(), testLogger())
			err := cmd.Execute(ctx, tc.userID, tc.articleHashID)
			require.NoError(t, err)
		})
	}
}

func TestRemoveArticleFromUserVector_Execute_FetchError(t *testing.T) {
	fetcher := mocks.NewMockArticleVectorFetcher(t)
	syncer := mocks.NewMockUserVectorSyncer(t)

	fetcher.EXPECT().
		FetchArticleVector(mock.Anything, "article1").
		Return(nil, errors.New("pinecone error"))

	cmd := &RemoveArticleFromUserVector{
		ArticleVectorFetcher: fetcher,
		UserVectorSyncer:     syncer,
	}

	ctx := domain.ContextWithLogger(context.Background(), testLogger())
	err := cmd.Execute(ctx, "user1", "article1")

	// Fetch errors are logged but not returned
	require.NoError(t, err)
}

func TestRemoveArticleFromUserVector_Execute_SubtractError(t *testing.T) {
	fetcher := mocks.NewMockArticleVectorFetcher(t)
	syncer := mocks.NewMockUserVectorSyncer(t)

	testVector := []float32{0.1}

	fetcher.EXPECT().
		FetchArticleVector(mock.Anything, "article1").
		Return(testVector, nil)

	syncer.EXPECT().
		SubtractArticleVectorFromUser(mock.Anything, "user1", "article1", testVector).
		Return(false, errors.New("db error"))

	cmd := &RemoveArticleFromUserVector{
		ArticleVectorFetcher: fetcher,
		UserVectorSyncer:     syncer,
	}

	ctx := domain.ContextWithLogger(context.Background(), testLogger())
	err := cmd.Execute(ctx, "user1", "article1")

	require.Error(t, err)
	require.Contains(t, err.Error(), "subtracting article vector from user")
}
