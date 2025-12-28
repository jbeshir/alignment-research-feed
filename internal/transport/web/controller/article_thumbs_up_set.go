package controller

import (
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
)

type ArticleThumbsUpSet struct {
	Fetcher        datasources.ArticleFetcher
	ThumbsUpSetter datasources.ArticleThumbsUpSetter
}

func (c ArticleThumbsUpSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handleFeedback(w, r, c.Fetcher, c.ThumbsUpSetter.SetArticleThumbsUp, "thumbs_up")
}
