\connect auth_system

INSERT INTO auth_users (id, username, display_name, email, status, created_at, updated_at)
VALUES (
    'usr_local_admin',
    'admin',
    'Local Demo Administrator',
    'admin@example.invalid',
    'active',
    now(),
    now()
)
ON CONFLICT (username) WHERE deleted_at IS NULL DO UPDATE
SET display_name = EXCLUDED.display_name,
    email = EXCLUDED.email,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO auth_credentials (
    id,
    user_id,
    credential_type,
    password_hash,
    password_hash_alg,
    password_hash_params_version,
    password_hash_params_json,
    password_changed_at,
    created_at,
    updated_at
)
VALUES (
    'cred_local_admin_password',
    'usr_local_admin',
    'password',
    '$argon2id$v=19$m=65536,t=3,p=2$czA1LWRlbW8tc2FsdC0wMQ$Oblehmj7QoR28oXyry/Y126GKnWdaqVnFgt4r64tVpw',
    'argon2id',
    'argon2id-v1',
    '{"memoryKiB":65536,"iterations":3,"parallelism":2,"saltBytes":16,"keyBytes":32}'::jsonb,
    now(),
    now(),
    now()
)
ON CONFLICT (user_id, credential_type) DO UPDATE
SET password_hash = EXCLUDED.password_hash,
    password_hash_alg = EXCLUDED.password_hash_alg,
    password_hash_params_version = EXCLUDED.password_hash_params_version,
    password_hash_params_json = EXCLUDED.password_hash_params_json,
    password_changed_at = now(),
    updated_at = now();

INSERT INTO user_roles (id, user_id, role_id, assigned_by, assigned_at, created_at)
SELECT 'urole_local_admin_super_admin', 'usr_local_admin', r.id, 'deploy-seed', now(), now()
FROM auth_roles r
WHERE r.code = 'super_admin'
ON CONFLICT (user_id, role_id) DO NOTHING;

\connect ai_gateway_system

INSERT INTO model_profiles (
    id,
    name,
    purpose,
    provider,
    base_url,
    model,
    enabled,
    is_default,
    timeout_ms,
    api_key_configured,
    supports_streaming,
    dimensions,
    top_n,
    default_parameters_json,
    created_by_user_id,
    updated_by_user_id
)
VALUES
    (
        'default-chat',
        'Local placeholder chat profile',
        'chat',
        'local_compatible',
        'http://localhost:11434/v1',
        'local-placeholder-chat',
        true,
        true,
        60000,
        false,
        true,
        NULL,
        NULL,
        '{"temperature":0.2}'::jsonb,
        'usr_local_admin',
        'usr_local_admin'
    ),
    (
        'default-embedding',
        'Local placeholder embedding profile',
        'embedding',
        'local_compatible',
        'http://localhost:11434/v1',
        'local-placeholder-embedding',
        true,
        true,
        60000,
        false,
        false,
        384,
        NULL,
        '{}'::jsonb,
        'usr_local_admin',
        'usr_local_admin'
    ),
    (
        'default-rerank',
        'Local placeholder rerank profile',
        'rerank',
        'local_compatible',
        'http://localhost:11434/v1',
        'local-placeholder-rerank',
        true,
        true,
        60000,
        false,
        false,
        NULL,
        10,
        '{}'::jsonb,
        'usr_local_admin',
        'usr_local_admin'
    )
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    base_url = EXCLUDED.base_url,
    model = EXCLUDED.model,
    enabled = EXCLUDED.enabled,
    is_default = EXCLUDED.is_default,
    timeout_ms = EXCLUDED.timeout_ms,
    api_key_configured = EXCLUDED.api_key_configured,
    supports_streaming = EXCLUDED.supports_streaming,
    dimensions = EXCLUDED.dimensions,
    top_n = EXCLUDED.top_n,
    default_parameters_json = EXCLUDED.default_parameters_json,
    updated_by_user_id = EXCLUDED.updated_by_user_id,
    updated_at = now();

\connect document_system

INSERT INTO report_types (code, name, description, enabled)
VALUES
    ('summer_peak_inspection', 'Summer Peak Inspection Report', 'Local demo report type for peak-season inspection workflows.', true),
    ('coal_inventory_audit', 'Coal Inventory Audit Report', 'Local demo report type for coal inventory audit workflows.', true)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    enabled = EXCLUDED.enabled,
    updated_at = now();

\connect knowledge_system

INSERT INTO knowledge_bases (
    id,
    name,
    description,
    doc_type,
    chunk_strategy,
    retrieval_strategy,
    created_by,
    created_at,
    updated_at
)
VALUES (
    'kb_local_demo',
    'Local Demo Knowledge Base',
    'Seed knowledge base for local integration smoke tests.',
    'GENERAL',
    '{"chunkSize":800,"overlap":120}'::jsonb,
    '{"topK":5,"scoreThreshold":0.2}'::jsonb,
    'usr_local_admin',
    now(),
    now()
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    doc_type = EXCLUDED.doc_type,
    chunk_strategy = EXCLUDED.chunk_strategy,
    retrieval_strategy = EXCLUDED.retrieval_strategy,
    updated_at = now(),
    deleted_at = NULL;
