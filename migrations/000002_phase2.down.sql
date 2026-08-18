ALTER TABLE environment_variables DROP CONSTRAINT IF EXISTS environment_variables_env_key_phase_unique;
ALTER TABLE environment_variables ADD CONSTRAINT environment_variables_environment_id_key_key UNIQUE (environment_id, key);
ALTER TABLE environment_variables
    DROP COLUMN IF EXISTS phase,
    DROP COLUMN IF EXISTS enabled,
    DROP COLUMN IF EXISTS type,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS expr;

ALTER TABLE collections
    DROP COLUMN IF EXISTS headers,
    DROP COLUMN IF EXISTS auth,
    DROP COLUMN IF EXISTS presets,
    DROP COLUMN IF EXISTS proxy,
    DROP COLUMN IF EXISTS client_certificates,
    DROP COLUMN IF EXISTS secrets;

ALTER TABLE workspaces DROP COLUMN IF EXISTS description;
