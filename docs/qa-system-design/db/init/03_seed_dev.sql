-- 可选：开发环境种子数据（仅本地 Docker 首次初始化时执行）
BEGIN;

INSERT INTO llm_config_versions (
    version_no, profile_id, model_name,
    timeout_seconds, temperature, max_tokens, is_active
) VALUES (
    1, 'gateway-default', 'gpt-4o-mini',
    60, 0.70, 4096, TRUE
);

INSERT INTO qa_config_versions (
    version_no, top_k, similarity_threshold, use_rerank, is_active, created_by_user_id
) VALUES (
    1, 5, 0.7000, FALSE, TRUE, 'dev-user-001'
);

COMMIT;
