-- Backfill nullable source_operation_id from older partial migrations
UPDATE requests SET source_operation_id = '' WHERE source_operation_id IS NULL;
ALTER TABLE requests ALTER COLUMN source_operation_id SET DEFAULT '';
ALTER TABLE requests ALTER COLUMN source_operation_id SET NOT NULL;
