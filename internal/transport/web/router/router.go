package router

import (
	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/transport/web/controller"
	"net/http"
	"time"
)

func MakeRouter(
	dataset datasources.DatasetRepository,
	similiarity datasources.SimilarityRepository,
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
	})

	r.Handle("/v1/articles/{article_id}", controller.ArticleGet{
		Fetcher:     dataset,
		CacheMaxAge: latestCacheMaxAge,
	})

	r.Handle("/v1/articles/{article_id}/similar", controller.SimilarArticlesList{
		Fetcher:     dataset,
		Similarity:  similiarity,
		CacheMaxAge: 0,
	})

	r.Handle("/v1/articles/{article_id}/mark-read", requireAuthMiddleware(controller.ArticleGet{
		Fetcher:     dataset,
		CacheMaxAge: latestCacheMaxAge,
	})).Methods(http.MethodPost)

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
