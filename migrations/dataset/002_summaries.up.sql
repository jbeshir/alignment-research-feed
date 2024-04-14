CREATE TABLE IF NOT EXISTS `summaries` (
    `id` int NOT NULL AUTO_INCREMENT,
    `text` text NOT NULL,
    `source` varchar(256) DEFAULT NULL,
    `article_id` int NOT NULL, PRIMARY KEY (`id`),
    KEY `article_id` (`article_id`),
    CONSTRAINT `summaries_ibfk_1` FOREIGN KEY (`article_id`) REFERENCES `articles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci