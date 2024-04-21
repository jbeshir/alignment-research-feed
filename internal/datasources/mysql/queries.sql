-- name: ListLatestArticles :many
SELECT hash_id, title, url, LEFT(COALESCE(text, ''), 500) as text_start, authors, date_published FROM articles
ORDER BY date_published DESC
LIMIT ?;