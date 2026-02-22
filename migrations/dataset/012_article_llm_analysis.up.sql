ALTER TABLE articles
    ADD COLUMN summary TEXT NULL,
    ADD COLUMN key_points TEXT NULL,
    ADD COLUMN implication TEXT NULL,
    ADD COLUMN category VARCHAR(128) NULL;
