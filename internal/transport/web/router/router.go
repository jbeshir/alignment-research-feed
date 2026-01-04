package router

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/command"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/transport/web/controller"
)

func MakeRouter(
	dataset datasources.DatasetRepository,
	similarity datasources.SimilarArticleLister,
	rssFeedBaseURL, rssFeedAuthorName, rssFeedAuthorEmail string,
	latestCacheMaxAge time.Duration,
	authMiddleware func(http.Handler) http.Handler,
) (http.Handler, error) {
	r := mux.NewRouter()
	r.Use(corsMiddleware)
	r.Use(authMiddleware)

	r.Handle("/v1/articles", controller.ArticlesList{
		Lister:      dataset,
		CacheMaxAge: latestCacheMaxAge,
	}).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/recommended", requireAuthMiddleware(controller.RecommendedArticlesList{
		Command: &command.RecommendArticles{
			ThumbsUpLister:   dataset,
			SimilarityLister: similarity,
			ArticleFetcher:   dataset,
		},
	})).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/{article_id}", controller.ArticleGet{
		Fetcher:     dataset,
		CacheMaxAge: latestCacheMaxAge,
	}).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/{article_id}/similar", controller.SimilarArticlesList{
		Fetcher:     dataset,
		Similarity:  similarity,
		CacheMaxAge: 0,
	}).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/{article_id}/read/{read}", requireAuthMiddleware(controller.ArticleReadSet{
		Fetcher:    dataset,
		ReadSetter: dataset,
	})).Methods(http.MethodPost, http.MethodOptions)

	r.Handle("/v1/articles/{article_id}/thumbs_up/{thumbs_up}", requireAuthMiddleware(controller.ArticleThumbsUpSet{
		Fetcher:        dataset,
		ThumbsUpSetter: dataset,
	})).Methods(http.MethodPost, http.MethodOptions)

	r.Handle("/v1/articles/{article_id}/thumbs_down/{thumbs_down}", requireAuthMiddleware(controller.ArticleThumbsDownSet{
		Fetcher:          dataset,
		ThumbsDownSetter: dataset,
	})).Methods(http.MethodPost, http.MethodOptions)

	rssFeeds := []controller.RSS{
		{
			FeedHostname:    rssFeedBaseURL,
			FeedPath:        "/rss",
			FeedAuthorName:  rssFeedAuthorName,
			FeedAuthorEmail: rssFeedAuthorEmail,
			Dataset:         dataset,
			CacheMaxAge:     latestCacheMaxAge,
		},
	}

	for _, feed := range rssFeeds {
		r.Handle(feed.FeedPath, feed)
	}

	return r, nil
}
