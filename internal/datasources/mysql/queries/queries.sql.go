// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: queries.sql

package queries

import (
	"context"
	"database/sql"
	"time"
)

const insertArticle = `-- name: InsertArticle :exec
INSERT INTO articles (
                      hash_id,
                      title,
                      url,
                      source,
                      text,
                      authors,
                      date_published,
                      date_created,
                      pinecone_status,
                      date_checked) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

type InsertArticleParams struct {
	HashID         string
	Title          sql.NullString
	Url            sql.NullString
	Source         sql.NullString
	Text           sql.NullString
	Authors        string
	DatePublished  sql.NullTime
	DateCreated    time.Time
	PineconeStatus string
	DateChecked    time.Time
}

func (q *Queries) InsertArticle(ctx context.Context, arg InsertArticleParams) error {
	_, err := q.db.ExecContext(ctx, insertArticle,
		arg.HashID,
		arg.Title,
		arg.Url,
		arg.Source,
		arg.Text,
		arg.Authors,
		arg.DatePublished,
		arg.DateCreated,
		arg.PineconeStatus,
		arg.DateChecked,
	)
	return err
}
