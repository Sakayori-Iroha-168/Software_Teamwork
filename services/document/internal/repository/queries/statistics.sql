-- name: GetReportStatisticsOverview :one
SELECT
    (SELECT COUNT(*) FROM report_templates WHERE deleted_at IS NULL)::int AS template_count,
    (SELECT COUNT(*) FROM reports WHERE deleted_at IS NULL)::int          AS report_count,
    (SELECT COUNT(*) FROM reports WHERE status = 'generated' AND deleted_at IS NULL)::int AS generated_count,
    (SELECT COUNT(*) FROM reports WHERE status = 'failed' AND deleted_at IS NULL)::int    AS failed_count;

-- name: GetReportDailyTrend :many
SELECT
    DATE(created_at) AS stat_date,
    COUNT(*)::int    AS generated_count
FROM reports
WHERE
    created_at >= NOW() - INTERVAL '30 days'
    AND deleted_at IS NULL
GROUP BY DATE(created_at)
ORDER BY stat_date ASC;
