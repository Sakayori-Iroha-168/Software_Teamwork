# QA 指标聚合接口

> 分支：`LiXiChuanTeam/feat/qa-metrics-aggregation`
> 日期：2026-06-29
> 状态：已实现

## 1. 目标

聚合问答总次数、今日问答数、平均延迟、今日活跃用户数、趋势数据点、热门问题、意图分布，提供稳定统计口径。

## 2. 接口

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| `GET` | `/api/v1/qa-metrics/overview` | `days`（默认 1） | 问答统计概览 |
| `GET` | `/api/v1/qa-metrics/trend` | `days`（默认 30） | 近 N 天问答趋势 |
| `GET` | `/api/v1/qa-metrics/top-queries` | `days`（默认 7）、`limit`（默认 10，最大 100） | 热门问题排行 |
| `GET` | `/api/v1/qa-metrics/intent-distribution` | `days`（默认 7） | 意图分布占比 |

## 3. 统计口径

### 3.1 数据来源

| 指标 | 来源表 | 过滤条件 |
|------|--------|----------|
| 问答总次数 | `response_runs` | `status IN ('completed','running')` |
| 今日问答数 | `response_runs` | `status IN ('completed','running') AND started_at >= current_date` |
| 平均延迟 | `response_runs` | `completed_at >= N天前 AND status='completed'` |
| 今日活跃用户 | `response_runs` 关联 `conversations` | `started_at >= current_date AND status IN ('completed','running')` |
| 总提问数 | `messages` | `role='user'` |
| 会话总数 | `conversations` | `deleted_at IS NULL` |
| 知识库数量 | knowledge 服务 | 暂未对接，默认 0 |
| 文档数量 | knowledge 服务 | 暂未对接，默认 0 |
| 趋势数据点 | `response_runs` | `status IN ('completed','running')`，按日聚合 |
| 热门问题 | `messages` 关联 `message_content_blocks` | 用户消息内容聚合 |
| 意图分布 | `response_runs.intent_type` | 空值归类为 `unknown` |

### 3.2 趋势完整日期补零

```sql
WITH dates AS (
  SELECT generate_series(current_date - ($1-1), current_date, '1 day')::date d
)
SELECT d::text, count(rr.id)
FROM dates
LEFT JOIN response_runs rr ON rr.started_at >= d AND rr.started_at < d+1
  AND rr.status IN ('completed','running')
GROUP BY d ORDER BY d
```

### 3.3 意图分布舍入

- 分母：所有意图类型的数量总和
- 舍入：`ROUND(数量 / 总数 * 1000) / 10`，保留一位小数
- 中文标签映射：`knowledge_qa`→知识问答、`general_chat`→一般对话、`report_generation`→报告生成、`data_analysis`→数据分析、其他→未知

## 4. 参数校验

| 参数 | 校验 |
|------|------|
| `days` | 必须为正整数，非法返回 `fields.days` 错误 |
| `limit` | 必须在 1 到 100 之间，超出返回 `fields.limit` 错误 |

## 5. 响应格式

成功响应：
```json
{"data": {...}, "requestId": "req_123"}
```

错误响应：
```json
{"error": {"code": "validation_error", "message": "请求参数错误", "requestId": "req_123", "fields": {...}}}
```

> 204 响应除外，无响应体。

## 6. 代码修改清单

| 文件 | 修改内容 |
|------|----------|
| `internal/service/resources.go` | `MetricsOverview` 结构体增加 `KnowledgeBaseCount` 和 `DocumentCount` 字段 |
| `internal/repository/resources_postgres.go` | 概览、趋势、意图分布的查询改用 `response_runs` 表统计，过滤有效运行状态 |

## 7. 验收标准

| 编号 | 标准 | 验证方式 |
|------|------|----------|
| 一 | 给定有数据、空数据和跨月日期窗口，当查询指标时，概览口径一致、趋势日期连续且空值不伪造为零 | `generate_series` 配合 `LEFT JOIN` 保证无数据日期补零 |
| 二 | 给定知识库指标获取失败或查询参数非法，当调用接口时，不可用与零值可区分，非法参数返回字段级错误 | 知识库/文档数量暂为 0 表示未对接；handler 层通过 `positiveQuery` 校验并返回字段错误 |

## 8. 依赖

- 2.1 意图写入 — `response_runs.intent_type` 由代理循环写入
- 功能六 数据仓储 — `response_runs` 表已有 `completed_at` 和 `started_at` 索引
- knowledge 服务 — 知识库数量和文档数量后续对接即可
- auth 服务 — handler 层已校验 `qa:settings:read` 管理员权限
