package domain

import "time"

// UserRatingType represents the type of rating a user gave to an article.
type UserRatingType string

const (
	// RatingTypeThumbsUp indicates a positive rating.
	RatingTypeThumbsUp UserRatingType = "thumbs_up"
	// RatingTypeThumbsDown indicates a negative rating.
	RatingTypeThumbsDown UserRatingType = "thumbs_down"
)

// UserArticleRating represents a stored vector for a user's rated article.
type UserArticleRating struct {
	ArticleHashID string
	Vector        []float32
	RatingType    UserRatingType
	RatedAt       time.Time
}
