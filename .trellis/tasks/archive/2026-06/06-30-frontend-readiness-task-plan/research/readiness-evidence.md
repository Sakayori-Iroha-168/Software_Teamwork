# Frontend readiness evidence for S-020

Date: 2026-06-30

## Issue baseline

- Issue: #264 `[S-020] 前端按后端就绪度重构任务体系提案`
- Assignee: `Jackeyliu37`
- Current branch: `Special/docs/frontend-readiness-task-plan`
- Scope: documentation and task planning only. Do not modify `apps/web/src`.

## F-series issue status

Source: `gh issue list --state all --search "F-001 ... F-012 in:title"`.

| Issue | Title | State | Interpretation for S-020 |
| --- | --- | --- | --- |
| #108 | `[F-001] 前端 OpenAPI 类型生成、typed client 与 SSE/上传封装` | Closed | L1/L2 foundation. Does not prove a complete business loop. |
| #109 | `[F-002] 前端登录、会话恢复、AppShell 与 RBAC 导航` | Closed | Mostly L1 shell plus auth integration. |
| #110 | `[F-003] 前端知识库、文档上传、处理状态与检索页面` | Closed | L1/L2 partial; retrieval and some document capabilities depend on Knowledge readiness. |
| #111 | `[F-004] 前端 Admin 模型 Profile 与解析器配置页面` | Closed | Model profile can be L2 where backend is ready; parser configs need gating until Knowledge implements active paths. |
| #112 | `[F-005] 前端智能问答聊天、SSE、工具摘要与引用展示` | Closed | L1/L2 partial; citations and true RAG depend on #93 and #84. |
| #113 | `[F-006] 前端 QA 配置、检索测试与统计页面` | Closed | L1/L2 partial; retrieval test depends on Knowledge retrieval readiness. |
| #114 | `[F-007] 前端报告模板、素材与报告记录页面` | Closed | L1/L2 partial; current pages still include fallback data. |
| #115 | `[F-008] 前端报告大纲、章节编辑、任务进度与导出流程` | Closed | L1/L2 partial; true AI generation is not proved by job status success. |
| #116 | `[F-009] 前端共享组件、状态覆盖与视觉一致性` | Closed | L1 quality and UI coverage. |
| #117 | `[F-010] 前端关键流程测试与 PR 前检查` | Open, PR #266 merged | Treat test baseline as present after PR #266, but business E2E against real backend is still separate. |
| #161 | `[F-011] 固定 openapi-typescript 并自动生成 Gateway 类型` | Closed | L2 foundation. |
| #162 | `[F-012] 前端 Gateway 类型落地与 legacy envelope 收敛` | Closed | L2 foundation; still needs feature-level readiness checks. |

## Backend and Gateway readiness

- Gateway active API owner map lists 97 active operations, including 18 Knowledge, 43 Document, and 25 QA operations.
- Frontend must generate clients only from `docs/services/gateway/api/openapi.yaml` and call only gateway `/api/v1/**`.
- Current missing public contracts are limited to admin overview and admin metrics aggregation.
- Knowledge implementation states multiple active Knowledge routes are still `NotImplemented`, especially document update/delete, chunks/content, `knowledge-queries`, and parser configs.
- QA implementation is route-complete for many QA paths, but true RAG still depends on Knowledge retrieval. Citations depend on #93 and Knowledge source availability.
- Document implementation currently includes report types/templates/materials/reports/outlines/sections, jobs/attempts/events, report files/content, basic in-process DOCX export, settings/statistics/logs, and operation logs. Remaining gaps are Document MCP tools, real AI outline/text generation, and rich DOCX generation through Pandoc/LibreOffice.
- Capability matrix still emphasizes cross-service smoke gaps. For frontend planning, a successful page render or job status transition must not be treated as complete L3 business acceptance.

## Frontend fallback and mock evidence

Source: `rg "fallback|mockRoutes|requestId" apps/web/src`.

- `apps/web/src/pages/reports/records/page.tsx` has `fallbackReports`.
- `apps/web/src/pages/reports/templates/page.tsx` has `fallbackTemplates` and `fallbackMaterials`.
- `apps/web/src/pages/reports/generate/page.tsx` has `fallbackTypes`, `fallbackTemplates`, `fallbackMaterials`, `fallbackOutline`, and `fallbackSections`.
- `apps/web/src/api/client.ts` has API-boundary `mockRoutes`, which are acceptable for tests but must not become production silent fallback.
- Admin QA pages already surface `requestId` in some error messages, so follow-up tasks can reuse that behavior.
- `apps/web/src/stores/auth-store.ts` has development mock user bypass. It is a dev convenience and must not count as real auth smoke.

## Related issue routing

- #84: Knowledge `knowledge-queries`, rerank, and MCP search contract are open/in progress. Knowledge and QA L2/L3 tasks must depend on it.
- #85: parser configs runtime configuration is open; admin parser config UI should capability-gate until ready.
- #93: QA citation snapshot/detail/batch lookup is open; QA citation UI should not fake citation details.
- #117: frontend critical-flow tests are covered by PR #266; they are API-boundary mocked tests, not real backend L3 smoke.
- #125: cross-service and MCP smoke is the right place for real gateway-service-tool smoke.
- #163: frontend test baseline is closed; do not duplicate baseline setup, only upgrade acceptance rules in a follow-up.

## Planning conclusion

S-020 should publish a proposal under `docs/` that:

- Reframes closed F tasks as L1 page skeleton or L2 partial API integration unless real backend readiness and smoke evidence exist.
- Defines L1/L2/L3 delivery boundaries.
- Drafts F-014, F-015, F-016, and S-022 issue briefs.
- Requires explicit capability gating for 501/not-ready paths.
- Forbids silent fallback as acceptance evidence.
