# 后端 · 问答统计接口 工作计划（DLQA-131 / DLQA-133）

- 负责人：阿荣毕力格（Ariunbileg，GitHub: `Aaka11x`）
- 团队：JerryTeam
- 范围：`services/qa` 下的管理后台统计接口
- 数据源：`response_runs`（统计事实表，**不新建按日统计主表**）

---

## DLQA-131 — `GET /api/admin/stats/overview`（聚合问答核心指标）

聚合指标：

| 指标 | 口径 |
|------|------|
| `total_qa` | `status='completed'` 的有效运行总数 |
| `today_qa` | 同上 + `started_at` 在今天（业务时区） |
| `avg_latency_ms` | `status='completed'` 且 `finished_at IS NOT NULL`（排除未完成运行） |
| `active_external_users` | `response_runs JOIN conversations` 的 `DISTINCT external_user_id` |
| `knowledge_base_count` / `document_count` | 调外部知识库统计接口（依赖 8.1.2）；失败标记 `"unavailable"`，**不伪造 0** |
| `generated_at` | 统计生成时间 |

> 状态：核心聚合 SQL 已在本地 PostgreSQL 用构造数据验证通过（0 与 unavailable 可区分、平均延迟排除未完成、distinct 用户正确）。

## DLQA-133 — `GET /api/admin/stats/trend?days=30`（近30天问答趋势）

- 按业务时区，从 `response_runs` 聚合最近 N 天每日问答次数。
- 服务端生成**完整日期序列**，缺失日期补 `0`。
- 返回 `{days, timezone, points: [[date, count], ...]}`；`points` 始终按日期升序，长度等于 `days`。
- 校验 `days` 合法范围；非法 `days` 返回错误码 **42200**。
- 跨月、零值、时区边界正确；查询只扫描 `response_runs`，不触及消息正文。

---

## 之后的工作如何开始（步骤）

1. 从最新 `develop` 创建分支：`JerryTeam/feat/qa-stats-overview`、`JerryTeam/feat/qa-stats-trend`。
2. 在 `services/qa` 实现 `overview` 接口（FastAPI + asyncpg），对齐团队 auth 与连接池实现。
3. 实现 `trend` 接口，封装“日期序列补 0”工具函数复用。
4. 补充单元测试；用本地 PostgreSQL（`qa-system-design/db`）联调。
5. 按 Conventional Commits 提交，向主仓库 `develop` 提 PR。

## 待确认

- 后端 Web 框架是否为 FastAPI；auth 与数据库连接池的接入方式。
- 外部知识库统计接口地址（依赖 8.1.2，由其他小组提供）。
- 请把我的 GitHub 账号 `Aaka11x` 加入 `.github/labeler.json` 的 `JerryTeam`，以便 PR 自动打标签。
