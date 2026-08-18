ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

ALTER TABLE collections
    ADD COLUMN IF NOT EXISTS headers JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS auth JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS presets JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS proxy JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS client_certificates JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS secrets JSONB NOT NULL DEFAULT '[]';

-- Migrate flat variables map to structured format
UPDATE collections
SET variables = jsonb_build_object(
    'pre_request', COALESCE(
        (SELECT jsonb_agg(jsonb_build_object(
            'enabled', true,
            'name', key,
            'value', value,
            'type', 'string',
            'description', '',
            'secret', false
        )) FROM jsonb_each_text(variables) AS t(key, value)),
        '[]'::jsonb
    ),
    'post_response', '[]'::jsonb
)
WHERE jsonb_typeof(variables) = 'object'
  AND NOT (variables ? 'pre_request');

ALTER TABLE collections ALTER COLUMN variables SET DEFAULT '{"pre_request":[],"post_response":[]}'::jsonb;

ALTER TABLE environment_variables
    ADD COLUMN IF NOT EXISTS phase TEXT NOT NULL DEFAULT 'pre_request',
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'string',
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS expr TEXT NOT NULL DEFAULT '';

ALTER TABLE environment_variables DROP CONSTRAINT IF EXISTS environment_variables_environment_id_key_key;
ALTER TABLE environment_variables ADD CONSTRAINT environment_variables_env_key_phase_unique UNIQUE (environment_id, key, phase);
