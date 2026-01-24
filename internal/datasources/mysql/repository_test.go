package mysql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/datasources/mysql/queries"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testArticleHash1 = "6a429bf5788aa30893172643f892fb74"
	testArticleHash2 = "59c45352ef0608a50c51afe9afbc23c3"
)

func setupTestDB(t *testing.T) *sql.DB {
	if testing.Short() {
		t.Skip("skipping MySQL integration tests in short mode")
	}

	db, err := Connect(context.Background(), os.Getenv("MYSQL_URI"))
	if err != nil {
		t.Fatal(err)
	}

	q := queries.New(db)

	err = q.InsertArticle(context.Background(), queries.InsertArticleParams{
		HashID:         testArticleHash1,
		Title:          sql.NullString{String: "Refusal in LLMs is mediated by a single direction", Valid: true},
		Url:            sql.NullString{String: "https://www.alignmentforum.org/posts/jGuXSZgv6qfdhMCuJ/refusal-in-llms-is-mediated-by-a-single-direction", Valid: true},
		Source:         sql.NullString{String: "alignmentforum", Valid: true},
		Text:           sql.NullString{String: "Post text 1", Valid: true},
		Authors:        "Andy Arditi,Oscar Obeso,Aaquib111,wesg,Neel Nanda",
		DatePublished:  sql.NullTime{Time: time.Date(2024, 4, 27, 11, 13, 6, 0, time.UTC), Valid: true},
		DateCreated:    time.Date(2024, 4, 28, 0, 27, 37, 0, time.UTC),
		PineconeStatus: "pending_addition",
		DateChecked:    time.Date(2024, 4, 28, 0, 27, 37, 0, time.UTC),
	})
	require.NoError(t, err)

	err = q.InsertArticle(context.Background(), queries.InsertArticleParams{
		HashID:         testArticleHash2,
		Title:          sql.NullString{String: "Constructability: Plainly-coded AGIs may be feasible in the near future", Valid: true},
		Url:            sql.NullString{String: "https://www.lesswrong.com/posts/y9tnz27oLmtLxcrEF/constructability-plainly-coded-agis-may-be-feasible-in-the", Valid: true},
		Source:         sql.NullString{String: "lesswrong", Valid: true},
		Text:           sql.NullString{String: "Post text 2", Valid: true},
		Authors:        "Épiphanie Gédéon,Charbel-Raphaël",
		DatePublished:  sql.NullTime{Time: time.Date(2024, 4, 27, 16, 04, 46, 0, time.UTC), Valid: true},
		DateCreated:    time.Date(2024, 4, 28, 0, 2, 13, 0, time.UTC),
		PineconeStatus: "pending_addition",
		DateChecked:    time.Date(2024, 4, 28, 0, 2, 13, 0, time.UTC),
	})
	require.NoError(t, err)

	// Set up some test ratings using SetArticleRead
	err = q.SetArticleRead(context.Background(), queries.SetArticleReadParams{
		ArticleHashID: testArticleHash1,
		UserID:        "test-user-123",
		HaveRead:      true,
	})
	require.NoError(t, err)

	return db
}

func teardownTestDB(t *testing.T, db *sql.DB) {
	if testing.Short() {
		t.Skip("skipping MySQL integration tests in short mode")
	}

	_, err := db.ExecContext(context.Background(), "DELETE FROM user_article_interactions")
	require.NoError(t, err)

	_, err = db.ExecContext(context.Background(), "DELETE FROM articles")
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
}

func TestRepository_ListLatestArticleIDs(t *testing.T) {

	cases := []struct {
		name     string
		filters  domain.ArticleFilters
		limit    int
		expected []string
	}{
		{
			name:    "all",
			filters: domain.ArticleFilters{},
			limit:   100,
			expected: []string{
				testArticleHash2,
				testArticleHash1,
			},
		},
		{
			name: "only_alignmentforum",
			filters: domain.ArticleFilters{
				SourcesAllowlist: []string{"alignmentforum"},
			},
			limit: 100,
			expected: []string{
				testArticleHash1,
			},
		},
		{
			name: "except_alignmentforum",
			filters: domain.ArticleFilters{
				SourcesBlocklist: []string{"alignmentforum"},
			},
			limit: 100,
			expected: []string{
				testArticleHash2,
			},
		},
	}

	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sut := New(db)

			results, err := sut.ListLatestArticleIDs(context.Background(), c.filters, domain.ArticleListOptions{
				PageSize: 100,
				Page:     1,
			})
			require.NoError(t, err)
			assert.Equal(t, c.expected, results)
		})
	}
}

func TestRepository_ListLatestArticleIDs_DateFilters(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	cases := []struct {
		name     string
		filters  domain.ArticleFilters
		expected []string
	}{
		{
			name: "published_after",
			filters: domain.ArticleFilters{
				PublishedAfter: time.Date(2024, 4, 27, 15, 0, 0, 0, time.UTC),
			},
			expected: []string{
				testArticleHash2, // Published at 16:04:46
			},
		},
		{
			name: "published_before",
			filters: domain.ArticleFilters{
				PublishedBefore: time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC),
			},
			expected: []string{
				testArticleHash1, // Published at 11:13:06
			},
		},
		{
			name: "published_between",
			filters: domain.ArticleFilters{
				PublishedAfter:  time.Date(2024, 4, 27, 10, 0, 0, 0, time.UTC),
				PublishedBefore: time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC),
			},
			expected: []string{
				testArticleHash1,
			},
		},
		{
			name: "no_matches",
			filters: domain.ArticleFilters{
				PublishedAfter: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: []string{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sut := New(db)

			results, err := sut.ListLatestArticleIDs(context.Background(), c.filters, domain.ArticleListOptions{
				PageSize: 100,
				Page:     1,
			})
			require.NoError(t, err)
			assert.Equal(t, c.expected, results)
		})
	}
}

func TestRepository_ListLatestArticleIDs_Ordering(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	cases := []struct {
		name     string
		options  domain.ArticleListOptions
		expected []string
	}{
		{
			name: "order_by_title_asc",
			options: domain.ArticleListOptions{
				PageSize: 100,
				Page:     1,
				Ordering: []domain.ArticleOrdering{
					{Field: domain.ArticleOrderingFieldTitle, Desc: false},
				},
			},
			expected: []string{
				testArticleHash2, // "Constructability..."
				testArticleHash1, // "Refusal in LLMs..."
			},
		},
		{
			name: "order_by_title_desc",
			options: domain.ArticleListOptions{
				PageSize: 100,
				Page:     1,
				Ordering: []domain.ArticleOrdering{
					{Field: domain.ArticleOrderingFieldTitle, Desc: true},
				},
			},
			expected: []string{
				testArticleHash1, // "Refusal in LLMs..."
				testArticleHash2, // "Constructability..."
			},
		},
		{
			name: "order_by_source",
			options: domain.ArticleListOptions{
				PageSize: 100,
				Page:     1,
				Ordering: []domain.ArticleOrdering{
					{Field: domain.ArticleOrderingFieldSource, Desc: false},
				},
			},
			expected: []string{
				testArticleHash1, // alignmentforum
				testArticleHash2, // lesswrong
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sut := New(db)

			results, err := sut.ListLatestArticleIDs(context.Background(), domain.ArticleFilters{}, c.options)
			require.NoError(t, err)
			assert.Equal(t, c.expected, results)
		})
	}
}

func TestRepository_ListLatestArticleIDs_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	cases := []struct {
		name         string
		page         int
		pageSize     int
		expectedLen  int
		expectedHash string
	}{
		{
			name:         "page_1_size_1",
			page:         1,
			pageSize:     1,
			expectedLen:  1,
			expectedHash: testArticleHash2,
		},
		{
			name:         "page_2_size_1",
			page:         2,
			pageSize:     1,
			expectedLen:  1,
			expectedHash: testArticleHash1,
		},
		{
			name:        "page_3_size_1_empty",
			page:        3,
			pageSize:    1,
			expectedLen: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sut := New(db)

			results, err := sut.ListLatestArticleIDs(context.Background(), domain.ArticleFilters{}, domain.ArticleListOptions{
				PageSize: c.pageSize,
				Page:     c.page,
			})
			require.NoError(t, err)
			assert.Len(t, results, c.expectedLen)
			if c.expectedLen > 0 {
				assert.Equal(t, c.expectedHash, results[0])
			}
		})
	}
}

func TestRepository_TotalMatchingArticles(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	cases := []struct {
		name     string
		filters  domain.ArticleFilters
		expected int64
	}{
		{
			name:     "all_articles",
			filters:  domain.ArticleFilters{},
			expected: 2,
		},
		{
			name: "only_lesswrong",
			filters: domain.ArticleFilters{
				SourcesAllowlist: []string{"lesswrong"},
			},
			expected: 1,
		},
		{
			name: "exclude_all",
			filters: domain.ArticleFilters{
				SourcesBlocklist: []string{"lesswrong", "alignmentforum"},
			},
			expected: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sut := New(db)

			count, err := sut.TotalMatchingArticles(context.Background(), c.filters)
			require.NoError(t, err)
			assert.Equal(t, c.expected, count)
		})
	}
}

func TestRepository_SetArticleRating(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	sut := New(db)
	ctx := context.Background()
	userID := "rating-test-user"
	articleID := testArticleHash1

	// Test SetArticleRead
	err := sut.SetArticleRead(ctx, articleID, userID, true)
	require.NoError(t, err)

	// Verify read status
	userCtx := domain.ContextWithUserID(ctx, userID)
	articles, err := sut.FetchArticlesByID(userCtx, []string{articleID})
	require.NoError(t, err)
	require.Len(t, articles, 1)
	require.NotNil(t, articles[0].HaveRead)
	assert.True(t, *articles[0].HaveRead)

	// Test SetArticleRating with thumbs up
	testVector := []float32{0.1, 0.2, 0.3}
	err = sut.SetArticleRating(ctx, userID, articleID, true, false, testVector)
	require.NoError(t, err)

	articles, err = sut.FetchArticlesByID(userCtx, []string{articleID})
	require.NoError(t, err)
	require.NotNil(t, articles[0].ThumbsUp)
	assert.True(t, *articles[0].ThumbsUp)

	// Test SetArticleRating with thumbs down
	err = sut.SetArticleRating(ctx, userID, articleID, false, true, testVector)
	require.NoError(t, err)

	articles, err = sut.FetchArticlesByID(userCtx, []string{articleID})
	require.NoError(t, err)
	require.NotNil(t, articles[0].ThumbsDown)
	assert.True(t, *articles[0].ThumbsDown)
	require.NotNil(t, articles[0].ThumbsUp)
	assert.False(t, *articles[0].ThumbsUp)

	// Test clearing rating
	err = sut.SetArticleRating(ctx, userID, articleID, false, false, testVector)
	require.NoError(t, err)

	articles, err = sut.FetchArticlesByID(userCtx, []string{articleID})
	require.NoError(t, err)
	require.NotNil(t, articles[0].ThumbsUp)
	assert.False(t, *articles[0].ThumbsUp)
	require.NotNil(t, articles[0].ThumbsDown)
	assert.False(t, *articles[0].ThumbsDown)
}

func TestRepository_ListThumbsUpArticleIDs(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	sut := New(db)
	ctx := context.Background()
	userID := "thumbs-up-test-user"

	// Initially no thumbs up
	ids, err := sut.ListThumbsUpArticleIDs(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, ids)

	// Add thumbs up to one article
	err = sut.SetArticleRating(ctx, userID, testArticleHash1, true, false, nil)
	require.NoError(t, err)

	ids, err = sut.ListThumbsUpArticleIDs(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, []string{testArticleHash1}, ids)

	// Add thumbs up to another article
	err = sut.SetArticleRating(ctx, userID, testArticleHash2, true, false, nil)
	require.NoError(t, err)

	ids, err = sut.ListThumbsUpArticleIDs(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, ids, 2)

	// Remove thumbs up
	err = sut.SetArticleRating(ctx, userID, testArticleHash1, false, false, nil)
	require.NoError(t, err)

	ids, err = sut.ListThumbsUpArticleIDs(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, []string{testArticleHash2}, ids)
}

func TestRepository_SetArticleRatingMultipleArticles(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	sut := New(db)
	ctx := context.Background()
	userID := "multi-vector-test-user"
	articleID1 := testArticleHash1
	articleID2 := testArticleHash2

	// Add vectors for both articles via SetArticleRating
	vector1 := []float32{1.0, 2.0, 3.0}
	vector2 := []float32{0.5, 1.0, 1.5}

	err := sut.SetArticleRating(ctx, userID, articleID1, true, false, vector1)
	require.NoError(t, err)

	err = sut.SetArticleRating(ctx, userID, articleID2, true, false, vector2)
	require.NoError(t, err)

	// Change one to thumbs down
	err = sut.SetArticleRating(ctx, userID, articleID1, false, true, vector1)
	require.NoError(t, err)

	// Verify ratings are correct
	ids, err := sut.ListThumbsUpArticleIDs(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, []string{articleID2}, ids)
}

func TestRepository_FetchArticlesByID(t *testing.T) {

	cases := []struct {
		name     string
		ids      []string
		userID   string
		expected []domain.Article
	}{
		{
			name:   "two_results_no_ratings",
			ids:    []string{testArticleHash2, testArticleHash1},
			userID: "",
			expected: []domain.Article{
				{
					HashID:      testArticleHash2,
					Title:       "Constructability: Plainly-coded AGIs may be feasible in the near future",
					Link:        "https://www.lesswrong.com/posts/y9tnz27oLmtLxcrEF/constructability-plainly-coded-agis-may-be-feasible-in-the",
					Source:      "lesswrong",
					TextStart:   "Post text 2",
					Authors:     "Épiphanie Gédéon,Charbel-Raphaël",
					PublishedAt: time.Date(2024, 4, 27, 16, 04, 46, 0, time.UTC),
				},
				{
					HashID:      testArticleHash1,
					Title:       "Refusal in LLMs is mediated by a single direction",
					Link:        "https://www.alignmentforum.org/posts/jGuXSZgv6qfdhMCuJ/refusal-in-llms-is-mediated-by-a-single-direction",
					Source:      "alignmentforum",
					TextStart:   "Post text 1",
					Authors:     "Andy Arditi,Oscar Obeso,Aaquib111,wesg,Neel Nanda",
					PublishedAt: time.Date(2024, 4, 27, 11, 13, 6, 0, time.UTC),
				},
			},
		},
		{
			name:   "one_result_no_ratings",
			ids:    []string{testArticleHash1, "does-not-exist"},
			userID: "",
			expected: []domain.Article{
				{
					HashID:      testArticleHash1,
					Title:       "Refusal in LLMs is mediated by a single direction",
					Link:        "https://www.alignmentforum.org/posts/jGuXSZgv6qfdhMCuJ/refusal-in-llms-is-mediated-by-a-single-direction",
					Source:      "alignmentforum",
					TextStart:   "Post text 1",
					Authors:     "Andy Arditi,Oscar Obeso,Aaquib111,wesg,Neel Nanda",
					PublishedAt: time.Date(2024, 4, 27, 11, 13, 6, 0, time.UTC),
				},
			},
		},
		{
			name:     "no_results",
			ids:      []string{"does-not-exist"},
			userID:   "",
			expected: []domain.Article{},
		},
		{
			name:   "with_ratings",
			ids:    []string{testArticleHash1},
			userID: "test-user-123",
			expected: []domain.Article{
				{
					HashID:      testArticleHash1,
					Title:       "Refusal in LLMs is mediated by a single direction",
					Link:        "https://www.alignmentforum.org/posts/jGuXSZgv6qfdhMCuJ/refusal-in-llms-is-mediated-by-a-single-direction",
					Source:      "alignmentforum",
					TextStart:   "Post text 1",
					Authors:     "Andy Arditi,Oscar Obeso,Aaquib111,wesg,Neel Nanda",
					PublishedAt: time.Date(2024, 4, 27, 11, 13, 6, 0, time.UTC),
					HaveRead:    func() *bool { b := true; return &b }(),
					ThumbsUp:    func() *bool { b := false; return &b }(),
					ThumbsDown:  func() *bool { b := false; return &b }(),
				},
			},
		},
	}

	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sut := New(db)

			ctx := t.Context()
			if c.userID != "" {
				ctx = domain.ContextWithUserID(ctx, c.userID)
			}

			results, err := sut.FetchArticlesByID(ctx, c.ids)
			require.NoError(t, err)
			assert.Equal(t, c.expected, results)
		})
	}
}
