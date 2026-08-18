ALTER TABLE api_specs DROP COLUMN IF EXISTS spec_content;

ALTER TABLE requests
    DROP COLUMN IF EXISTS docs_overridden,
    DROP COLUMN IF EXISTS api_doc;
