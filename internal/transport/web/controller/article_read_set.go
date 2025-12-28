package controller

import (
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
)

type ArticleReadSet struct {
	Fetcher    datasources.ArticleFetcher
	ReadSetter datasources.ArticleReadSetter
}

func (c ArticleReadSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handleFeedback(w, r, c.Fetcher, c.ReadSetter.SetArticleRead, "read")
}
