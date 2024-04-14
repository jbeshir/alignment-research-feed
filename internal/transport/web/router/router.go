package router

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/transport/web/controller"
	"net/http"
)

func MakeRouter(
	ctx context.Context,
	dataset datasources.DatasetRepository,
	rssFeedBaseURL, rssFeedAuthorName, rssFeedAuthorEmail string,
) (http.Handler, error) {
	r := mux.NewRouter()

	rssFeeds := []controller.RSS{
		{
			FeedHostname:    rssFeedBaseURL,
			FeedPath:        "/rss",
			FeedAuthorName:  rssFeedAuthorName,
			FeedAuthorEmail: rssFeedAuthorEmail,
			Dataset:         dataset,
		},
	}

	for _, feed := range rssFeeds {
		r.Handle(feed.FeedPath, feed)
	}

	return r, nil
}
