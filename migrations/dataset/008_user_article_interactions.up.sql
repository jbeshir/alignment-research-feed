CREATE TABLE IF NOT EXISTS `user_article_interactions` (
    `user_id` VARCHAR(256) NOT NULL,
    `article_hash_id` VARCHAR(32) NOT NULL,
    `have_read` BOOLEAN NOT NULL DEFAULT FALSE,
    `thumbs_up` BOOLEAN NOT NULL DEFAULT FALSE,
    `thumbs_down` BOOLEAN NOT NULL DEFAULT FALSE,
    `date_read` DATETIME DEFAULT NULL,
    `date_rated` DATETIME DEFAULT NULL,
    `vector` LONGBLOB DEFAULT NULL,
    PRIMARY KEY (`user_id`, `article_hash_id`),
    INDEX idx_user_unreviewed (`user_id`, `have_read`, `thumbs_up`, `thumbs_down`, `date_read`),
    INDEX idx_user_rated (`user_id`, `date_rated`),
    INDEX idx_user_thumbs_up (`user_id`, `thumbs_up`, `date_rated`),
    INDEX idx_user_thumbs_down (`user_id`, `thumbs_down`, `date_rated`),
    CONSTRAINT `user_article_interactions_ibfk_1` FOREIGN KEY (`article_hash_id`)
        REFERENCES `articles` (`hash_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
