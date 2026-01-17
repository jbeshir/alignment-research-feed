-- User recommendation vectors table (stores sum, not average - divide by count when using)
CREATE TABLE IF NOT EXISTS `user_recommendation_vectors` (
    `user_id` VARCHAR(256) NOT NULL,
    `vector_sum` LONGBLOB NOT NULL,
    `vector_count` INT NOT NULL DEFAULT 0,
    `updated_at` DATETIME NOT NULL,
    PRIMARY KEY (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Track whether each rating's vector has been added to the user's sum
ALTER TABLE `article_ratings`
    ADD COLUMN `vector_added` BOOLEAN NOT NULL DEFAULT FALSE;
