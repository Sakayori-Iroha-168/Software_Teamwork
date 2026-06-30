# S-020 frontend readiness task plan

## Goal

Create a public planning document for issue #264 that reclassifies existing frontend work by backend readiness, proposes a three-layer frontend delivery model, and drafts follow-up issues for Report, Knowledge, QA Chat, and cross-module frontend smoke acceptance.

## What I already know

* Issue #264 is assigned to `Jackeyliu37` and has been explicitly taken over in the issue comments.
* This is a documentation/planning task, not an `apps/web/src` implementation task.
* The proposal must use `docs/` as the source of truth when code or old local drafts conflict.
* The task must reference gateway active routes, service implementation documents, and the current capability matrix.
* Required deliverables are:
  * `F-014 Report 前端去除 fallback 并接入真实 Document API`
  * `F-015 Knowledge 前端按 Gateway 501 状态做 capability gating`
  * `F-016 QA 聊天引用与工具展示对齐后端能力`
  * `S-022 前端跨模块 E2E smoke 与 issue 验收标准升级`
  * A PM-facing recommendation that explains which closed F-series issues were L1 page skeletons, which need L2 real API follow-up, and which content belongs in #125 or #163.

## Requirements

* Add a proposal document under `docs/` that is discoverable from the collaboration or architecture documentation set.
* Explain the frontend three-layer delivery model:
  * L1 page skeleton and interaction shell.
  * L2 real API integration according to backend readiness.
  * L3 cross-module E2E smoke against gateway and capability matrix.
* Reclassify F-001 through F-012 as UI skeleton, partial real API integration, or not proven as a complete business loop.
* Compare frontend-facing pages with Gateway / backend implementation readiness, especially:
  * gateway paths still returning 501.
  * workers or cross-service runtime paths not ready.
  * placeholder data, mock-only flows, or silent fallback.
* Draft follow-up issue briefs using the repository issue format and Chinese task titles.
* Explicitly prohibit silent fallback from satisfying frontend acceptance.
* Do not directly modify frontend application source code.

## Acceptance Criteria

* [ ] The proposal clearly states L1/L2/L3 frontend delivery definitions and acceptance boundaries.
* [ ] Each proposed follow-up issue cites concrete backend readiness evidence from Gateway OpenAPI, capability matrix, or service implementation docs.
* [ ] Report follow-up covers removal of `fallbackTypes`, `fallbackTemplates`, `fallbackReports` and error/requestId handling.
* [ ] Knowledge follow-up covers capability gating for not-ready or 501 routes while preserving real active CRUD/upload flows.
* [ ] QA Chat follow-up covers citations, tool summaries, RAG degradation, and avoids mock thinking/tool steps.
* [ ] Frontend smoke follow-up covers login, admin pages, QA chat, and report workspace acceptance rules.
* [ ] The document explains what should be included in #125 and #163 instead of duplicating those tasks.
* [ ] `git diff --check` passes.

## Definition of Done

* Proposal document added or updated in `docs/`.
* Relevant docs index updated if a new document is added.
* Trellis task context is recorded.
* Validation commands are run and recorded before commit/PR.
* PR body includes `Closes #264`.

## Technical Approach

Use a documentation-only implementation:

* Read the authoritative docs named in issue #264.
* Inspect frontend source only to identify existing fallback/mock/capability gaps; do not edit app code.
* Write one clear proposal document with:
  * current state summary.
  * L1/L2/L3 model.
  * readiness gap matrix.
  * follow-up issue drafts.
  * PM-facing recommendations.
* Link the document from `docs/README.md` or another relevant docs entry point if needed.

## Decision (ADR-lite)

**Context**: Existing closed frontend issues may be misunderstood as complete business workflows, while backend routes and service implementation readiness are still uneven.

**Decision**: Treat #264 as a planning and governance document. Do not reopen old F-series issues and do not change frontend runtime code here. Create new follow-up issue drafts that bind frontend work to backend readiness evidence.

**Consequences**: The PR stays small and reviewable. Real UI code changes remain in future F-series tasks, and cross-module smoke remains aligned with #125 / #163.

## Out of Scope

* No changes to `apps/web/src/`.
* No direct creation of GitHub follow-up issues unless the user asks after the proposal is reviewed.
* No implementation of missing backend 501 routes.
* No replacement for #117, #125, or #163.

## Technical Notes

Authoritative references from issue #264:

* #262
* `docs/collaboration/task-brief-template.md`
* `docs/collaboration/task-issue-project-workflow.md`
* `docs/collaboration/frontend-workflow.md`
* `.trellis/spec/frontend/index.md`
* `.trellis/spec/frontend/quality-guidelines.md`
* `docs/architecture/current-capability-matrix.md`
* `docs/services/gateway/api/openapi.yaml`
* `docs/services/knowledge/docs/implementation.md`
* `docs/services/qa/docs/implementation.md`
* `docs/services/document/docs/implementation.md`
* #84 #85 #93 #117 #125 #163
