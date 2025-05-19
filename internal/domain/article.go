package domain

import (
	"time"
)

type Article struct {
	HashID      string    `json:"hash_id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	TextStart   string    `json:"text_start"`
	Authors     string    `json:"authors"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`

	HaveRead   *bool `json:"have_read,omitempty"`
	ThumbsUp   *bool `json:"thumbs_up,omitempty"`
	ThumbsDown *bool `json:"thumbs_down,omitempty"`
}

type ArticleListMetadata struct {
	TotalRows int `json:"total_rows"`
}

type ArticleFilters struct {
	SourcesAllowlist []string
	SourcesBlocklist []string
	PublishedAfter   time.Time
	PublishedBefore  time.Time
	TitleFulltext    string
	AuthorsFulltext  string
}

type ArticleListOptions struct {
	Limit          int64
	Ordering       []ArticleOrdering
	Page, PageSize int
}

type ArticleOrdering struct {
	Field ArticleOrderingField
	Desc  bool
}

type SimilarArticle struct {
	HashID string
	Score  float64
}

type ArticleOrderingField string

const ArticleOrderingFieldPublishedAt ArticleOrderingField = "published_at"
const ArticleOrderingFieldAuthors ArticleOrderingField = "authors"
const ArticleOrderingFieldSource ArticleOrderingField = "source"
const ArticleOrderingFieldTitle ArticleOrderingField = "title"

var ValidOrderingFields = []ArticleOrderingField{
	ArticleOrderingFieldPublishedAt,
	ArticleOrderingFieldAuthors,
	ArticleOrderingFieldSource,
	ArticleOrderingFieldTitle,
}
