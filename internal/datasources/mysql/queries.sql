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
    date_published,
    have_read,
    thumbs_up,
    thumbs_down
FROM articles
LEFT JOIN article_ratings
    ON articles.hash_id = article_ratings.article_hash_id
        AND article_ratings.user_id = ?
WHERE hash_id IN (sqlc.slice('hash_ids'));

-- name: SetArticleRead :exec
INSERT INTO article_ratings (
        article_hash_id,
        user_id,
        have_read,
        thumbs_up,
        thumbs_down
    ) VALUES (?, ?, sqlc.arg(have_read), FALSE, FALSE)
ON DUPLICATE KEY UPDATE have_read = sqlc.arg(have_read);

-- name: SetArticleThumbsUp :exec
INSERT INTO article_ratings (
        article_hash_id,
        user_id,
        have_read,
        thumbs_up,
        thumbs_down
    ) VALUES (?, ?, FALSE, sqlc.arg(thumbs_up), FALSE)
ON DUPLICATE KEY UPDATE
    thumbs_up = sqlc.arg(thumbs_up),
    thumbs_down = IF(sqlc.arg(thumbs_up), FALSE, thumbs_down);

-- name: SetArticleThumbsDown :exec
INSERT INTO article_ratings (
        article_hash_id,
        user_id,
        have_read,
        thumbs_up,
        thumbs_down
    ) VALUES (?, ?, FALSE, FALSE, sqlc.arg(thumbs_down))
ON DUPLICATE KEY UPDATE
    thumbs_down = sqlc.arg(thumbs_down),
    thumbs_up = IF(sqlc.arg(thumbs_down), FALSE, thumbs_up);
