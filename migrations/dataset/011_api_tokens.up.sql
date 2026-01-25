CREATE TABLE IF NOT EXISTS `api_tokens` (
    `id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(256) NOT NULL,
    `token_hash` CHAR(64) NOT NULL,
    `token_prefix` CHAR(8) NOT NULL,
    `name` VARCHAR(128) DEFAULT NULL,
    `created_at` DATETIME NOT NULL DEFAULT NOW(),
    `last_used_at` DATETIME DEFAULT NULL,
    `expires_at` DATETIME DEFAULT NULL,
    `revoked_at` DATETIME DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_token_hash` (`token_hash`),
    INDEX `idx_user_id` (`user_id`),
    INDEX `idx_user_active` (`user_id`, `revoked_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
