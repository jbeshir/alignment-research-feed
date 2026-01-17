-- For unreviewed articles query (read but not rated)
CREATE INDEX idx_article_ratings_user_unreviewed
    ON article_ratings (user_id, have_read, thumbs_up, thumbs_down, date_read);

-- For reviewed articles query (rated)
CREATE INDEX idx_article_ratings_user_reviewed
    ON article_ratings (user_id, date_reviewed);
