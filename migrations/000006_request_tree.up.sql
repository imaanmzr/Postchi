ALTER TABLE requests
    ADD COLUMN IF NOT EXISTS template_id UUID REFERENCES requests(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS overridden_fields TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX idx_requests_template ON requests(template_id);
