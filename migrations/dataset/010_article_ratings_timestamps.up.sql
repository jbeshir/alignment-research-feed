ALTER TABLE `article_ratings`
    ADD COLUMN `date_read` DATETIME DEFAULT NULL,
    ADD COLUMN `date_reviewed` DATETIME DEFAULT NULL;

-- Backfill existing data with NOW() (all existing rows get same timestamp)
UPDATE article_ratings SET date_read = NOW() WHERE have_read = TRUE AND date_read IS NULL;
UPDATE article_ratings SET date_reviewed = NOW() WHERE (thumbs_up = TRUE OR thumbs_down = TRUE) AND date_reviewed IS NULL;
