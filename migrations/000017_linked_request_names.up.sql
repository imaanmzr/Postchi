ALTER TABLE workspace_docs
    ADD COLUMN IF NOT EXISTS linked_request_names TEXT[] NOT NULL DEFAULT '{}';
