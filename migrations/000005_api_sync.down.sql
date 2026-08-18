ALTER TABLE requests DROP CONSTRAINT IF EXISTS fk_requests_source_spec;
DROP TABLE IF EXISTS api_spec_environment_urls;
DROP TABLE IF EXISTS api_specs;
ALTER TABLE requests DROP COLUMN IF EXISTS source_spec_id;
ALTER TABLE requests DROP COLUMN IF EXISTS source_operation_id;
ALTER TABLE requests DROP COLUMN IF EXISTS source_op_hash;
ALTER TABLE environments DROP COLUMN IF EXISTS stage;
