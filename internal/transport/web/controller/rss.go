package controller

import (
	"fmt"
	"github.com/gorilla/feeds"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
	"time"
)

type RSS struct {
	FeedHostname    string
	FeedPath        string
	FeedAuthorName  string
	FeedAuthorEmail string
	Dataset         datasources.DatasetRepository
	CacheMaxAge     time.Duration
}

func (c RSS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	feed := &feeds.Feed{
		Title:       "Alignment Research Feed",
		Link:        &feeds.Link{Href: c.FeedHostname + c.FeedPath},
		Description: "Feed of new papers and posts added to the alignment research dataset",
		Author:      &feeds.Author{Name: c.FeedAuthorName, Email: c.FeedAuthorEmail},
		Created:     time.Now(),
	}

	articles, err := c.Dataset.ListLatestArticles(r.Context(), 100)
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to fetch articles for feed", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, a := range articles {
		feed.Items = append(feed.Items, &feeds.Item{
			Id:          a.HashID,
			IsPermaLink: "false",
			Title:       a.Title,
			Link:        &feeds.Link{Href: a.Link},
			Description: a.TextStart,
			Author: &feeds.Author{
				Name: a.Authors,
			},
			Created: a.Published,
		})
	}

	rss, err := feed.ToRss()
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to format feed as RSS", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(c.CacheMaxAge.Seconds())))

	if _, err := w.Write([]byte(rss)); err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to write feed to response", "error", err)
	}
}
