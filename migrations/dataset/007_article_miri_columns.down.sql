-- Revert pinecone_status back to VARCHAR
ALTER TABLE articles MODIFY COLUMN pinecone_status VARCHAR(32) NOT NULL;

-- Remove comment from confidence column
ALTER TABLE articles MODIFY COLUMN confidence FLOAT;

-- Drop the new MIRI-related columns
ALTER TABLE articles DROP COLUMN needs_tech;

ALTER TABLE articles DROP COLUMN miri_distance;

ALTER TABLE articles DROP COLUMN miri_confidence;