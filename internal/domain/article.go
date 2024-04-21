package domain

import "time"

type Article struct {
	HashID    string
	Title     string
	Link      string
	TextStart string
	Authors   string
	Published time.Time
}
