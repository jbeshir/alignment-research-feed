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
    summary,
    key_points,
    implication,
    category,
    have_read,
    thumbs_up,
    thumbs_down
FROM articles
LEFT JOIN user_article_interactions
    ON articles.hash_id = user_article_interactions.article_hash_id
        AND user_article_interactions.user_id = ?
WHERE hash_id IN (sqlc.slice('hash_ids'));

-- name: ListThumbsUpArticleIDs :many
SELECT article_hash_id FROM user_article_interactions
WHERE user_id = ? AND thumbs_up = TRUE;

-- name: SetArticleRead :exec
INSERT INTO user_article_interactions (
        user_id,
        article_hash_id,
        have_read,
        thumbs_up,
        thumbs_down,
        date_read
    ) VALUES (?, ?, sqlc.arg(have_read), FALSE, FALSE, sqlc.arg(date_read))
ON DUPLICATE KEY UPDATE
    have_read = sqlc.arg(have_read),
    date_read = COALESCE(date_read, sqlc.arg(date_read));

-- name: ListUnreviewedArticleIDs :many
SELECT article_hash_id FROM user_article_interactions
WHERE user_id = ?
    AND have_read = TRUE
    AND thumbs_up = FALSE
    AND thumbs_down = FALSE
ORDER BY date_read DESC
LIMIT ? OFFSET ?;

-- name: ListLikedArticleIDs :many
SELECT article_hash_id FROM user_article_interactions
WHERE user_id = ? AND thumbs_up = TRUE
ORDER BY date_rated DESC
LIMIT ? OFFSET ?;

-- name: ListDislikedArticleIDs :many
SELECT article_hash_id FROM user_article_interactions
WHERE user_id = ? AND thumbs_down = TRUE
ORDER BY date_rated DESC
LIMIT ? OFFSET ?;

-- name: ListReadArticleIDs :many
SELECT article_hash_id FROM user_article_interactions
WHERE user_id = ? AND have_read = TRUE;

-- ============================================
-- User Article Interactions (unified table)
-- ============================================

-- name: GetUserArticleInteraction :one
SELECT user_id, article_hash_id, have_read, thumbs_up, thumbs_down,
       date_read, date_rated, `vector`
FROM user_article_interactions
WHERE user_id = ? AND article_hash_id = ?;

-- name: UpsertUserArticleInteraction :exec
INSERT INTO user_article_interactions (
    user_id, article_hash_id, have_read, thumbs_up, thumbs_down,
    date_read, date_rated, `vector`
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    have_read = VALUES(have_read),
    thumbs_up = VALUES(thumbs_up),
    thumbs_down = VALUES(thumbs_down),
    date_read = VALUES(date_read),
    date_rated = VALUES(date_rated),
    `vector` = VALUES(`vector`);

-- name: GetUserArticleVectorsByThumbsUp :many
SELECT article_hash_id, `vector`, date_rated
FROM user_article_interactions
WHERE user_id = ? AND thumbs_up = TRUE AND `vector` IS NOT NULL
ORDER BY date_rated DESC;

-- name: GetUserArticleVectorsByThumbsDown :many
SELECT article_hash_id, `vector`, date_rated
FROM user_article_interactions
WHERE user_id = ? AND thumbs_down = TRUE AND `vector` IS NOT NULL
ORDER BY date_rated DESC;

-- name: CountUserArticleVectorsByThumbsUp :one
SELECT COUNT(*) as count
FROM user_article_interactions
WHERE user_id = ? AND thumbs_up = TRUE AND `vector` IS NOT NULL;

-- name: CountUserArticleVectorsByThumbsDown :one
SELECT COUNT(*) as count
FROM user_article_interactions
WHERE user_id = ? AND thumbs_down = TRUE AND `vector` IS NOT NULL;

-- ============================================
-- User Interest Clusters
-- ============================================

-- name: UpsertUserInterestCluster :exec
INSERT INTO user_interest_clusters (user_id, cluster_id, centroid_vector, article_count, updated_at)
VALUES (?, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
    centroid_vector = VALUES(centroid_vector),
    article_count = VALUES(article_count),
    updated_at = NOW();

-- name: GetUserInterestClusters :many
SELECT cluster_id, centroid_vector, article_count, updated_at
FROM user_interest_clusters
WHERE user_id = ?
ORDER BY cluster_id;

-- name: DeleteUserInterestClusters :exec
DELETE FROM user_interest_clusters
WHERE user_id = ?;

-- ============================================
-- Precomputed Recommendations
-- ============================================

-- name: UpsertPrecomputedRecommendation :exec
INSERT INTO user_precomputed_recommendations (user_id, article_hash_id, score, source, position, generated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    score = VALUES(score),
    source = VALUES(source),
    position = VALUES(position),
    generated_at = VALUES(generated_at);

-- name: DeleteUserPrecomputedRecommendations :exec
DELETE FROM user_precomputed_recommendations
WHERE user_id = ?;

-- name: GetPrecomputedRecommendations :many
SELECT article_hash_id, score, source, position, generated_at
FROM user_precomputed_recommendations
WHERE user_id = ?
ORDER BY position ASC
LIMIT ?;

-- name: GetPrecomputedRecommendationAge :one
SELECT generated_at
FROM user_precomputed_recommendations
WHERE user_id = ?
ORDER BY generated_at DESC
LIMIT 1;

-- ============================================
-- User Recommendation State
-- ============================================

-- name: GetUserRecommendationState :one
SELECT last_generated_at, last_rating_at, needs_regeneration
FROM user_recommendation_state
WHERE user_id = ?;

-- name: UpsertUserRecommendationState :exec
INSERT INTO user_recommendation_state (user_id, last_generated_at, last_rating_at, needs_regeneration)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    last_generated_at = COALESCE(VALUES(last_generated_at), last_generated_at),
    last_rating_at = COALESCE(VALUES(last_rating_at), last_rating_at),
    needs_regeneration = VALUES(needs_regeneration);

-- name: MarkUserNeedsRegeneration :exec
INSERT INTO user_recommendation_state (user_id, last_rating_at, needs_regeneration)
VALUES (?, NOW(), TRUE)
ON DUPLICATE KEY UPDATE
    last_rating_at = NOW(),
    needs_regeneration = TRUE;

-- name: MarkUserRegenerated :exec
UPDATE user_recommendation_state
SET last_generated_at = NOW(), needs_regeneration = FALSE
WHERE user_id = ?;

-- name: ListUsersNeedingRegeneration :many
SELECT user_id
FROM user_recommendation_state
WHERE needs_regeneration = TRUE
ORDER BY last_rating_at ASC;

-- ============================================
-- API Tokens
-- ============================================

-- name: CreateAPIToken :exec
INSERT INTO api_tokens (id, user_id, token_hash, token_prefix, name, created_at, expires_at)
VALUES (?, ?, ?, ?, ?, NOW(), ?);

-- name: GetAPITokenByHash :one
SELECT id, user_id, token_hash, token_prefix, name, created_at, last_used_at, expires_at, revoked_at
FROM api_tokens
WHERE token_hash = ?;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_tokens
SET last_used_at = NOW()
WHERE id = ?;

-- name: ListUserAPITokens :many
SELECT id, user_id, token_hash, token_prefix, name, created_at, last_used_at, expires_at, revoked_at
FROM api_tokens
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: CountUserActiveAPITokens :one
SELECT COUNT(*) as count
FROM api_tokens
WHERE user_id = ? AND revoked_at IS NULL;

-- name: RevokeAPIToken :exec
UPDATE api_tokens
SET revoked_at = NOW()
WHERE id = ? AND user_id = ?;
