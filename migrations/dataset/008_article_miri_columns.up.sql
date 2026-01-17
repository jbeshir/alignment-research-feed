-- Add new MIRI-related columns
ALTER TABLE articles ADD COLUMN miri_confidence FLOAT COMMENT 'How much MIRI wants this in the chatbot';

ALTER TABLE articles ADD COLUMN miri_distance VARCHAR(128) NULL COMMENT 'Whether this is core or wider from MIRI\'s perspective';

ALTER TABLE articles ADD COLUMN needs_tech BOOLEAN NULL COMMENT 'Whether the article is about technical details';

-- Update existing column comments and types
ALTER TABLE articles MODIFY COLUMN confidence FLOAT COMMENT 'Describes the confidence in how good this article is, as a value <0, 1>';

-- Create enum type for pinecone_status and update column
ALTER TABLE articles MODIFY COLUMN pinecone_status ENUM('absent', 'pending_removal', 'pending_addition', 'added') NOT NULL;