CREATE TABLE IF NOT EXISTS `article_ratings` (
    `article_hash_id` VARCHAR(32) NOT NULL,
    `user_id` VARCHAR(256) NOT NULL,
    `have_read` BOOLEAN,
    `thumbs_up` BOOLEAN,
    `thumbs_down` BOOLEAN,
    PRIMARY KEY (`article_hash_id`, `user_id`),
    CONSTRAINT `article_ratings_ibfk_1` FOREIGN KEY (`article_hash_id`) REFERENCES `articles` (`hash_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci