DROP INDEX IF EXISTS idx_requests_template;
ALTER TABLE requests DROP COLUMN IF EXISTS template_id;
ALTER TABLE requests DROP COLUMN IF EXISTS is_template;
ALTER TABLE requests DROP COLUMN IF EXISTS overridden_fields;
