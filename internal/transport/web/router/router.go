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
) (http.Handler, error) {
	r := mux.NewRouter()
	r.Use(corsMiddleware)

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
