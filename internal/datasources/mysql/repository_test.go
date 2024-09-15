package mysql

import (
	"context"
	"database/sql"
	"github.com/jbeshir/alignment-research-feed/internal/datasources/mysql/queries"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
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

	return db
}

func teardownTestDB(t *testing.T, db *sql.DB) {
	if testing.Short() {
		t.Skip("skipping MySQL integration tests in short mode")
	}

	_, err := db.ExecContext(context.Background(), "DELETE FROM articles")
	assert.NoError(t, err)

	err = db.Close()
	assert.NoError(t, err)
}

func TestRepository_ListLatestArticles(t *testing.T) {

	cases := []struct {
		name     string
		filters  domain.ArticleFilters
		limit    int
		expected []domain.Article
	}{
		{
			name:    "all",
			filters: domain.ArticleFilters{},
			limit:   100,
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
			name: "only_alignmentforum",
			filters: domain.ArticleFilters{
				SourcesAllowlist: []string{"alignmentforum"},
			},
			limit: 100,
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
			name: "except_alignmentforum",
			filters: domain.ArticleFilters{
				SourcesBlocklist: []string{"alignmentforum"},
			},
			limit: 100,
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
			},
		},
	}

	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sut := New(db)

			results, err := sut.ListLatestArticles(context.Background(), c.filters, domain.ArticleListOptions{
				PageSize: 100,
				Page:     1,
			})
			assert.NoError(t, err)
			assert.Equal(t, c.expected, results)
		})
	}
}

func TestRepository_FetchArticlesByID(t *testing.T) {

	cases := []struct {
		name     string
		ids      []string
		expected []domain.Article
	}{
		{
			name: "two_results",
			ids:  []string{"59c45352ef0608a50c51afe9afbc23c3", "6a429bf5788aa30893172643f892fb74"},
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
			name: "one_result",
			ids:  []string{"6a429bf5788aa30893172643f892fb74", "does-not-exist"},
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
			expected: []domain.Article{},
		},
	}

	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sut := New(db)

			results, err := sut.FetchArticlesByID(context.Background(), c.ids)
			assert.NoError(t, err)
			assert.Equal(t, c.expected, results)
		})
	}
}
