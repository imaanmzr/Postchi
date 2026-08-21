CREATE TYPE workspace_type AS ENUM ('default', 'pm', 'tester');

ALTER TABLE workspaces
    ADD COLUMN type workspace_type NOT NULL DEFAULT 'default';
