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

-- name: ListThumbsUpArticleIDs :many
SELECT article_hash_id FROM article_ratings
WHERE user_id = ? AND thumbs_up = TRUE;

-- name: SetArticleRead :exec
INSERT INTO article_ratings (
        article_hash_id,
        user_id,
        have_read,
        thumbs_up,
        thumbs_down,
        date_read
    ) VALUES (?, ?, sqlc.arg(have_read), FALSE, FALSE, sqlc.arg(date_read))
ON DUPLICATE KEY UPDATE
    have_read = sqlc.arg(have_read),
    date_read = COALESCE(date_read, sqlc.arg(date_read));

-- name: SetArticleThumbsUp :exec
INSERT INTO article_ratings (
        article_hash_id,
        user_id,
        have_read,
        thumbs_up,
        thumbs_down,
        date_reviewed
    ) VALUES (?, ?, FALSE, sqlc.arg(thumbs_up), FALSE, sqlc.arg(date_reviewed))
ON DUPLICATE KEY UPDATE
    thumbs_up = sqlc.arg(thumbs_up),
    thumbs_down = IF(sqlc.arg(thumbs_up), FALSE, thumbs_down),
    date_reviewed = COALESCE(date_reviewed, sqlc.arg(date_reviewed));

-- name: SetArticleThumbsDown :exec
INSERT INTO article_ratings (
        article_hash_id,
        user_id,
        have_read,
        thumbs_up,
        thumbs_down,
        date_reviewed
    ) VALUES (?, ?, FALSE, FALSE, sqlc.arg(thumbs_down), sqlc.arg(date_reviewed))
ON DUPLICATE KEY UPDATE
    thumbs_down = sqlc.arg(thumbs_down),
    thumbs_up = IF(sqlc.arg(thumbs_down), FALSE, thumbs_up),
    date_reviewed = COALESCE(date_reviewed, sqlc.arg(date_reviewed));

-- name: GetUserVector :one
SELECT vector_sum, vector_count FROM user_recommendation_vectors WHERE user_id = ?;

-- name: UpsertUserVectorAdd :exec
INSERT INTO user_recommendation_vectors (user_id, vector_sum, vector_count, updated_at)
VALUES (?, ?, 1, NOW())
ON DUPLICATE KEY UPDATE
    vector_sum = ?,
    vector_count = vector_count + 1,
    updated_at = NOW();

-- name: UpdateUserVectorSubtract :exec
UPDATE user_recommendation_vectors
SET vector_sum = ?, vector_count = vector_count - 1, updated_at = NOW()
WHERE user_id = ?;

-- name: GetRatingVectorAdded :one
SELECT vector_added FROM article_ratings WHERE user_id = ? AND article_hash_id = ?;

-- name: SetRatingVectorAdded :exec
UPDATE article_ratings SET vector_added = ? WHERE user_id = ? AND article_hash_id = ?;

-- name: ListUnreviewedArticleIDs :many
SELECT article_hash_id FROM article_ratings
WHERE user_id = ?
    AND have_read = TRUE
    AND (thumbs_up = FALSE OR thumbs_up IS NULL)
    AND (thumbs_down = FALSE OR thumbs_down IS NULL)
ORDER BY date_read DESC
LIMIT ? OFFSET ?;

-- name: ListLikedArticleIDs :many
SELECT article_hash_id FROM article_ratings
WHERE user_id = ? AND thumbs_up = TRUE
ORDER BY date_reviewed DESC
LIMIT ? OFFSET ?;

-- name: ListDislikedArticleIDs :many
SELECT article_hash_id FROM article_ratings
WHERE user_id = ? AND thumbs_down = TRUE
ORDER BY date_reviewed DESC
LIMIT ? OFFSET ?;
