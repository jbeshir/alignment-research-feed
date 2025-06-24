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
		HashID:         "6a429bf5788aa30893172643f892fb74",
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
		HashID:         "59c45352ef0608a50c51afe9afbc23c3",
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

	// Set up some test ratings
	err = q.SetArticleRead(context.Background(), queries.SetArticleReadParams{
		ArticleHashID: "6a429bf5788aa30893172643f892fb74",
		UserID:        "test-user-123",
		HaveRead:      sql.NullBool{Bool: true, Valid: true},
	})
	require.NoError(t, err)

	return db
}

func teardownTestDB(t *testing.T, db *sql.DB) {
	if testing.Short() {
		t.Skip("skipping MySQL integration tests in short mode")
	}

	_, err := db.ExecContext(context.Background(), "DELETE FROM article_ratings")
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
				"59c45352ef0608a50c51afe9afbc23c3",
				"6a429bf5788aa30893172643f892fb74",
			},
		},
		{
			name: "only_alignmentforum",
			filters: domain.ArticleFilters{
				SourcesAllowlist: []string{"alignmentforum"},
			},
			limit: 100,
			expected: []string{
				"6a429bf5788aa30893172643f892fb74",
			},
		},
		{
			name: "except_alignmentforum",
			filters: domain.ArticleFilters{
				SourcesBlocklist: []string{"alignmentforum"},
			},
			limit: 100,
			expected: []string{
				"59c45352ef0608a50c51afe9afbc23c3",
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

func TestRepository_FetchArticlesByID(t *testing.T) {

	cases := []struct {
		name     string
		ids      []string
		userID   string
		expected []domain.Article
	}{
		{
			name:   "two_results_no_ratings",
			ids:    []string{"59c45352ef0608a50c51afe9afbc23c3", "6a429bf5788aa30893172643f892fb74"},
			userID: "",
			expected: []domain.Article{
				{
					HashID:      "59c45352ef0608a50c51afe9afbc23c3",
					Title:       "Constructability: Plainly-coded AGIs may be feasible in the near future",
					Link:        "https://www.lesswrong.com/posts/y9tnz27oLmtLxcrEF/constructability-plainly-coded-agis-may-be-feasible-in-the",
					Source:      "lesswrong",
					TextStart:   "Post text 2",
					Authors:     "Épiphanie Gédéon,Charbel-Raphaël",
					PublishedAt: time.Date(2024, 4, 27, 16, 04, 46, 0, time.UTC),
				},
				{
					HashID:      "6a429bf5788aa30893172643f892fb74",
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
			ids:    []string{"6a429bf5788aa30893172643f892fb74", "does-not-exist"},
			userID: "",
			expected: []domain.Article{
				{
					HashID:      "6a429bf5788aa30893172643f892fb74",
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
			ids:    []string{"6a429bf5788aa30893172643f892fb74"},
			userID: "test-user-123",
			expected: []domain.Article{
				{
					HashID:      "6a429bf5788aa30893172643f892fb74",
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
