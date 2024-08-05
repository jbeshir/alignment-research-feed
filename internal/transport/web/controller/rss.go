package controller

import (
	"fmt"
	"github.com/gorilla/feeds"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
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

	filters, err := articleFiltersFromQuery(r.URL.Query())
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to parse article filters in query string", "error", err)

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	options, err := listOptionsFromQuery(r.URL.Query())
	if err != nil {
		ctx := r.Context()
		logger := domain.LoggerFromContext(ctx)
		logger.ErrorContext(ctx, "unable to parse article list options in query string", "error", err)

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	articles, err := c.Dataset.ListLatestArticles(r.Context(), filters, options)
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
			Created: a.PublishedAt,
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

func articleFiltersFromQuery(q url.Values) (domain.ArticleFilters, error) {
	var filters domain.ArticleFilters

	if q.Has("only_sources") {
		filters.OnlySources = strings.Split(q.Get("only_sources"), ",")
	}

	if q.Has("except_sources") {
		filters.ExceptSources = strings.Split(q.Get("except_sources"), ",")
	}

	return filters, nil
}

func listOptionsFromQuery(q url.Values) (domain.ArticleListOptions, error) {
	var options domain.ArticleListOptions
	if q.Has("page") {
		page, err := strconv.ParseInt(q.Get("page"), 10, 32)
		if err != nil {
			return domain.ArticleListOptions{}, fmt.Errorf("unable to parse page from query: %w", err)
		}
		if page < 1 {
			return domain.ArticleListOptions{}, fmt.Errorf("invalid page value [%d]", page)
		}
		options.Page = int(page)
	} else {
		options.Page = 1
	}

	if q.Has("page_size") {
		pageSize, err := strconv.ParseInt(q.Get("page_size"), 10, 32)
		if err != nil {
			return domain.ArticleListOptions{}, fmt.Errorf("unable to parse page size from query: %w", err)
		}
		if pageSizeLimit := int64(200); pageSize > pageSizeLimit {
			return domain.ArticleListOptions{}, fmt.Errorf("page size [%d] exceeds limit [%d]",
				pageSize, pageSizeLimit)
		}
		options.PageSize = int(pageSize)
	} else {
		options.PageSize = 100
	}

	if q.Has("sort") {
		orderings := strings.Split(q.Get("sort"), ",")

		for _, ordering := range orderings {
			field := ordering
			desc := false
			if strings.HasSuffix(ordering, "_desc") {
				field = ordering[:len(ordering)-5]
				desc = true
			}

			if !slices.Contains(domain.ValidOrderingFields, domain.ArticleOrderingField(field)) {
				return domain.ArticleListOptions{}, fmt.Errorf("unrecognised article ordering field: %s", field)
			}

			options.Ordering = append(options.Ordering, domain.ArticleOrdering{
				Field: domain.ArticleOrderingField(field),
				Desc:  desc,
			})
		}
	}

	return options, nil
}
