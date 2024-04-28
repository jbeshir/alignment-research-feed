package domain

import (
	"time"
)

type Article struct {
	HashID    string
	Title     string
	Link      string
	TextStart string
	Authors   string
	Source    string
	Published time.Time
}

type ArticleFilters struct {
	OnlySources   []string
	ExceptSources []string
}
