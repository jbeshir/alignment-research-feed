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
}

type ArticleFilters struct {
	OnlySources   []string
	ExceptSources []string
}
