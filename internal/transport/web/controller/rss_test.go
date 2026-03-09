package controller

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jbeshir/alignment-research-feed/internal/datasources/mocks"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRSSDataset struct {
	*mocks.LatestArticleLister
	*mocks.ArticleFetcher
}

func newRSSController(t *testing.T) (RSS, *mocks.LatestArticleLister, *mocks.ArticleFetcher) {
	lister := mocks.NewLatestArticleLister(t)
	fetcher := mocks.NewArticleFetcher(t)

	c := RSS{
		FeedHostname:    "https://example.com",
		FeedPath:        "/rss",
		FeedAuthorName:  "Test Author",
		FeedAuthorEmail: "test@example.com",
		Dataset: &mockRSSDataset{
			LatestArticleLister: lister,
			ArticleFetcher:      fetcher,
		},
		CacheMaxAge: time.Hour,
	}
	return c, lister, fetcher
}

func TestRSS_ServeHTTP(t *testing.T) {
	pubTime := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	t.Run("produces valid RSS XML with category", func(t *testing.T) {
		controller, lister, fetcher := newRSSController(t)

		lister.EXPECT().
			ListLatestArticleIDs(mock.Anything, mock.Anything, mock.Anything).
			Return([]string{"h1", "h2"}, nil)
		fetcher.EXPECT().
			FetchArticlesByID(mock.Anything, []string{"h1", "h2"}).
			Return([]domain.Article{
				{
					HashID:      "h1",
					Title:       "Interpretability Research",
					Link:        "https://example.com/article1",
					Authors:     "Alice",
					Summary:     "A summary of interpretability work",
					Category:    "Interpretability",
					PublishedAt: &pubTime,
				},
				{
					HashID:      "h2",
					Title:       "Governance Update",
					Link:        "https://example.com/article2",
					Authors:     "Bob",
					TextStart:   "Some text start fallback",
					Category:    "Governance & Policy",
					PublishedAt: &pubTime,
				},
			}, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/rss", nil)
		req = testContext()(req)
		rec := httptest.NewRecorder()

		controller.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/xml", rec.Header().Get("Content-Type"))
		assert.Equal(t, "max-age=3600", rec.Header().Get("Cache-Control"))

		body := rec.Body.String()

		// Verify it's valid XML by parsing it
		require.True(t, strings.HasPrefix(body, xml.Header), "should start with XML header")
		var parsed struct {
			XMLName xml.Name `xml:"rss"`
			Channel struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				Items       []struct {
					Title       string `xml:"title"`
					Link        string `xml:"link"`
					Description string `xml:"description"`
					Category    string `xml:"category"`
					Author      string `xml:"author"`
					Guid        struct {
						Value       string `xml:",chardata"`
						IsPermaLink string `xml:"isPermaLink,attr"`
					} `xml:"guid"`
					PubDate string `xml:"pubDate"`
				} `xml:"item"`
			} `xml:"channel"`
		}
		err := xml.Unmarshal([]byte(body), &parsed)
		require.NoError(t, err, "response should be valid XML")

		// Channel-level checks
		assert.Equal(t, "Alignment Research Feed", parsed.Channel.Title)
		assert.Equal(t, "https://example.com/rss", parsed.Channel.Link)

		// Item checks
		require.Len(t, parsed.Channel.Items, 2)

		item1 := parsed.Channel.Items[0]
		assert.Equal(t, "Interpretability Research", item1.Title)
		assert.Equal(t, "https://example.com/article1", item1.Link)
		assert.Equal(t, "A summary of interpretability work", item1.Description)
		assert.Equal(t, "Interpretability", item1.Category)
		assert.Equal(t, "h1", item1.Guid.Value)
		assert.Equal(t, "false", item1.Guid.IsPermaLink)
		assert.NotEmpty(t, item1.PubDate)

		item2 := parsed.Channel.Items[1]
		assert.Equal(t, "Governance Update", item2.Title)
		assert.Equal(t, "Some text start fallback", item2.Description, "should fall back to TextStart when Summary is empty")
		assert.Equal(t, "Governance & Policy", item2.Category)
	})

	t.Run("empty category omitted from XML", func(t *testing.T) {
		controller, lister, fetcher := newRSSController(t)

		lister.EXPECT().
			ListLatestArticleIDs(mock.Anything, mock.Anything, mock.Anything).
			Return([]string{"h1"}, nil)
		fetcher.EXPECT().
			FetchArticlesByID(mock.Anything, []string{"h1"}).
			Return([]domain.Article{
				{
					HashID:    "h1",
					Title:     "No Category Article",
					Link:      "https://example.com/a",
					TextStart: "text",
				},
			}, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/rss", nil)
		req = testContext()(req)
		rec := httptest.NewRecorder()

		controller.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "<category>")
	})

	t.Run("empty feed produces valid RSS", func(t *testing.T) {
		controller, lister, fetcher := newRSSController(t)

		lister.EXPECT().
			ListLatestArticleIDs(mock.Anything, mock.Anything, mock.Anything).
			Return([]string{}, nil)
		fetcher.EXPECT().
			FetchArticlesByID(mock.Anything, []string{}).
			Return([]domain.Article{}, nil)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/rss", nil)
		req = testContext()(req)
		rec := httptest.NewRecorder()

		controller.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "<channel>")
		assert.NotContains(t, rec.Body.String(), "<item>")
	})

	t.Run("list IDs error returns 500", func(t *testing.T) {
		controller, lister, _ := newRSSController(t)

		lister.EXPECT().
			ListLatestArticleIDs(mock.Anything, mock.Anything, mock.Anything).
			Return(nil, assert.AnError)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/rss", nil)
		req = testContext()(req)
		rec := httptest.NewRecorder()

		controller.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("fetch articles error returns 500", func(t *testing.T) {
		controller, lister, fetcher := newRSSController(t)

		lister.EXPECT().
			ListLatestArticleIDs(mock.Anything, mock.Anything, mock.Anything).
			Return([]string{"h1"}, nil)
		fetcher.EXPECT().
			FetchArticlesByID(mock.Anything, []string{"h1"}).
			Return(nil, assert.AnError)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/rss", nil)
		req = testContext()(req)
		rec := httptest.NewRecorder()

		controller.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("invalid filter returns 400", func(t *testing.T) {
		controller, _, _ := newRSSController(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/rss?filter_published_after=not-a-date", nil)
		req = testContext()(req)
		rec := httptest.NewRecorder()

		controller.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestArticleDescription(t *testing.T) {
	t.Run("prefers summary", func(t *testing.T) {
		a := domain.Article{Summary: "AI summary", TextStart: "Raw text"}
		assert.Equal(t, "AI summary", articleDescription(a))
	})

	t.Run("falls back to text start", func(t *testing.T) {
		a := domain.Article{TextStart: "Raw text"}
		assert.Equal(t, "Raw text", articleDescription(a))
	})

	t.Run("empty when both empty", func(t *testing.T) {
		a := domain.Article{}
		assert.Empty(t, articleDescription(a))
	})
}
