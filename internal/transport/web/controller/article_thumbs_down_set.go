package controller

import (
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
)

type ArticleThumbsDownSet struct {
	Fetcher          datasources.ArticleFetcher
	ThumbsDownSetter datasources.ArticleThumbsDownSetter
}

func (c ArticleThumbsDownSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handleFeedback(w, r, c.Fetcher, c.ThumbsDownSetter.SetArticleThumbsDown, "thumbs_down")
}
