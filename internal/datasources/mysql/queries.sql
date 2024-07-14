-- name: InsertArticle :exec
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
                      date_checked) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListLatestArticles :many
SELECT hash_id, title, url, source, LEFT(COALESCE(text, ''), 500) as text_start, authors, date_published FROM articles
WHERE
    (0 = sqlc.arg('only_sources_filter') OR source IN (sqlc.slice('only_sources')))
    AND (0 = sqlc.arg('except_sources_filter') OR source NOT IN (sqlc.slice('except_sources')))
ORDER BY date_published DESC
LIMIT ? OFFSET ?;

-- name: TotalMatchingArticles :one
SELECT COUNT(*) FROM articles
WHERE
    (0 = sqlc.arg('only_sources_filter') OR source IN (sqlc.slice('only_sources')))
  AND (0 = sqlc.arg('except_sources_filter') OR source NOT IN (sqlc.slice('except_sources')))
ORDER BY date_published DESC;