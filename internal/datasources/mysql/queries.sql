-- name: ListLatestArticles :many
SELECT title, url, LEFT(text, 500) as text_start, authors, date_published FROM articles
ORDER BY date_published
LIMIT ?;