DROP INDEX `idx_article_ratings_vector_pending` ON `article_ratings`;
ALTER TABLE `article_ratings` DROP COLUMN `vector_added`;
DROP TABLE IF EXISTS `user_recommendation_vectors`;
