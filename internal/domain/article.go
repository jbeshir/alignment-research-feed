package domain

import "time"

type Article struct {
	Title     string
	Link      string
	TextStart string
	Authors   string
	Published time.Time
}
