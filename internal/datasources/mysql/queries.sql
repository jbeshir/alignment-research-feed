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

-- name: FetchArticlesByID :many
SELECT
    hash_id,
    title,
    url,
    source,
    LEFT(COALESCE(text, ''), 500) as text_start,
    authors,
    date_published
FROM articles
WHERE hash_id IN (sqlc.slice('hash_ids'))