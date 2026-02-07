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
	similarity datasources.SimilarityRepository,
	embedder datasources.Embedder,
	rssFeedBaseURL, rssFeedAuthorName, rssFeedAuthorEmail string,
	latestCacheMaxAge time.Duration,
	authMiddleware func(http.Handler) http.Handler,
	createAPITokenCmd *command.CreateAPIToken,
	recommendArticlesCmd *command.RecommendArticles,
) (http.Handler, error) {
	r := mux.NewRouter()
	r.Use(corsMiddleware)
	r.Use(authMiddleware)

	// Create shared command for rating updates
	setRatingCmd := command.NewSetArticleRating(similarity, dataset, dataset)

	r.Handle("/v1/articles", controller.ArticlesList{
		Lister:      dataset,
		CacheMaxAge: latestCacheMaxAge,
	}).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/recommended", requireAuthMiddleware(controller.RecommendedArticlesList{
		Command: recommendArticlesCmd,
	})).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/unreviewed", requireAuthMiddleware(controller.UserArticlesList{
		Fetcher:    dataset,
		ListFunc:   dataset.ListUnreviewedArticleIDs,
		ListEntity: "unreviewed",
	})).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/liked", requireAuthMiddleware(controller.UserArticlesList{
		Fetcher:    dataset,
		ListFunc:   dataset.ListLikedArticleIDs,
		ListEntity: "liked",
	})).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/disliked", requireAuthMiddleware(controller.UserArticlesList{
		Fetcher:    dataset,
		ListFunc:   dataset.ListDislikedArticleIDs,
		ListEntity: "disliked",
	})).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/articles/semantic-search", controller.SemanticSearch{
		Embedder:   embedder,
		Similarity: similarity,
		Fetcher:    dataset,
	}).Methods(http.MethodPost, http.MethodOptions)

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

	r.Handle("/v1/articles/{article_id}/thumbs_up/{thumbs_up}", requireAuthMiddleware(controller.ArticleRatingSet{
		Fetcher:      dataset,
		SetRatingCmd: setRatingCmd,
		RatingType:   controller.RatingTypeThumbsUp,
	})).Methods(http.MethodPost, http.MethodOptions)

	r.Handle("/v1/articles/{article_id}/thumbs_down/{thumbs_down}", requireAuthMiddleware(controller.ArticleRatingSet{
		Fetcher:      dataset,
		SetRatingCmd: setRatingCmd,
		RatingType:   controller.RatingTypeThumbsDown,
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

	// API Token management endpoints (no API token auth allowed)
	r.Handle("/v1/tokens", requireNonAPITokenAuthMiddleware(controller.APITokenCreate{
		CreateCmd: createAPITokenCmd,
	})).Methods(http.MethodPost, http.MethodOptions)

	r.Handle("/v1/tokens", requireNonAPITokenAuthMiddleware(controller.APITokenList{
		TokenLister: dataset,
	})).Methods(http.MethodGet, http.MethodOptions)

	r.Handle("/v1/tokens/{token_id}", requireNonAPITokenAuthMiddleware(controller.APITokenRevoke{
		TokenRevoker: dataset,
	})).Methods(http.MethodDelete, http.MethodOptions)

	return r, nil
}
