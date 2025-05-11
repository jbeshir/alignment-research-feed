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
WHERE hash_id IN (sqlc.slice('hash_ids'));

-- name: MarkArticleRead :exec
INSERT INTO article_ratings (
        article_hash_id,
        user_id,
        have_read,
        thumbs_up,
        thumbs_down
    ) VALUES (?, ?, TRUE, FALSE, FALSE)
ON DUPLICATE KEY UPDATE have_read = TRUE;
