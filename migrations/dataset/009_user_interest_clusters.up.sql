-- Store user interest cluster centroids for multi-interest recommendations
-- Users may have 3-5 distinct interest areas within alignment research
CREATE TABLE IF NOT EXISTS `user_interest_clusters` (
    `user_id` VARCHAR(256) NOT NULL,
    `cluster_id` INT NOT NULL,
    `centroid_vector` LONGBLOB NOT NULL,
    `article_count` INT NOT NULL DEFAULT 0,
    `updated_at` DATETIME NOT NULL,
    PRIMARY KEY (`user_id`, `cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
