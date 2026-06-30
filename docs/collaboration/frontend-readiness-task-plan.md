# 前端按后端就绪度重构任务体系提案

日期：2026-06-30
关联任务：GitHub Issue #264 `[S-020] 前端按后端就绪度重构任务体系提案`

本文用于把既有前端任务从“页面已完成”重新校准为“按后端就绪度分层交付”。它不重开 F-001 到 F-012，不替代 #117、#125 或 #163，也不直接修改 `apps/web/src`。

## 目标

当前前端已经完成了一批页面骨架、Gateway 类型、API client、测试基线和局部真实接口接入。但后端能力并非全部进入完整业务闭环：部分 Gateway active path 仍可能由下游返回 501，部分 worker 只完成状态流转，部分页面仍存在 fallback 数据。为了避免把“页面能打开”误判为“业务闭环完成”，后续前端任务需要按 L1/L2/L3 分层验收。

权威依据包括：

- `docs/architecture/current-capability-matrix.md`
- `docs/architecture/frontend-backend-contract.md`
- `docs/services/gateway/api/openapi.yaml`
- `docs/services/gateway/docs/active-api-owner-map.md`
- `docs/services/document/docs/implementation.md`
- `docs/services/knowledge/docs/implementation.md`
- `docs/services/qa/docs/implementation.md`
- `.trellis/spec/frontend/index.md`
- `.trellis/spec/frontend/quality-guidelines.md`

## 三层交付模型

| 层级 | 定义 | 可以验收的内容 | 不能当作验收的内容 |
| --- | --- | --- | --- |
| L1 页面骨架 | 页面、路由、导航、权限可见性、空/加载/错误状态和主要交互 shell 已存在 | 页面可访问；组件状态完整；mock 位于测试或显式开发边界 | 使用假数据、fallback 或 mock 后直接宣称业务完成 |
| L2 真实 API 接入 | 前端按 Gateway OpenAPI 和后端 readiness 接入真实 `/api/v1/**`，并处理 envelope、requestId、权限和 501/未就绪状态 | 已 ready endpoint 的真实请求、失败提示、requestId 展示、能力 gating | 后端未 ready 时静默 fallback；绕过 Gateway 调内部服务 |
| L3 跨模块 E2E smoke | 通过 Gateway 串联 Auth、Knowledge、QA、Document、File、AI Gateway 或 MCP 工具，完成可重复的业务 smoke | 脚本或清单证明真实链路可跑；未就绪能力显式 skipped/expected failure | 单页组件测试、API-boundary mock 测试或单服务状态机成功 |

后续 issue 标题和验收必须写明目标层级。关闭 L1 issue 只表示页面与交互骨架可用，不自动表示 L2 或 L3 已完成。

## 当前 readiness 结论

| 模块 | 当前可依赖能力 | 主要缺口 | 前端处理建议 |
| --- | --- | --- | --- |
| Gateway | OpenAPI 和 active owner map 已列出 97 个 active operation；前端只调用 `/api/v1/**` | 部分 Knowledge active path 的下游实现仍可能返回 501；admin overview/metrics 聚合仍是 missing contract | 所有业务请求走 Gateway；active path 也要按实际 readiness 做错误和能力状态 |
| Report / Document | 报告类型、模板、素材、报告、outline、section、jobs/events、report files/content、基础 DOCX、settings/statistics/logs 在当前代码和 implementation 中可见 | Document MCP tools、真实 AI 大纲/正文生成、Pandoc/LibreOffice 富 DOCX、跨服务 File+Redis+worker smoke 仍需补齐 | 去掉报告页面 silent fallback；基础 CRUD 和文件导出可做 L2；AI 生成和 MCP 工具不得假完成 |
| Knowledge | 知识库 CRUD、文档上传 handoff、PostgreSQL repository 和 asynq enqueue 已实现 | document PATCH/DELETE、chunks/content、`knowledge-queries`、parser configs、worker ingestion、Parser/Qdrant/embedding/rerank 闭环仍未完全 ready | 已 ready 的 CRUD/upload 继续真实 API；未 ready 或 501 路径做 capability gating |
| QA Chat | QA sessions/messages/SSE/config/resource routes 基本存在；模型调用走 AI Gateway 方向一致 | true RAG 依赖 #84；引用详情依赖 #93；跨服务 smoke 未证明完整闭环 | 聊天可做基础消息/SSE L2；引用、工具摘要、RAG 结果要按后端能力展示，不造假步骤 |
| Frontend tests | #163 已关闭；#117 对应 PR #266 已提供 API-boundary mocked critical-flow tests | 这类测试不是真实后端 L3 smoke；#117 issue 状态可能仍需 Project/Issue 人工收口 | 保留 mock 测试作为回归；新增 S-022 定义真实 readiness smoke 验收 |

## F-001 到 F-012 重新定位

| Issue | 状态 | 重新定位 | 后续处理 |
| --- | --- | --- | --- |
| #108 F-001 OpenAPI 类型生成、typed client 与 SSE/上传封装 | Closed | L2 基础设施 | 继续作为真实 API 接入基础，不代表任何业务闭环已完成 |
| #109 F-002 登录、会话恢复、AppShell 与 RBAC 导航 | Closed | L1 shell + 部分 L2 auth | 可纳入 S-022 登录 smoke，不需要重开 |
| #110 F-003 知识库、文档上传、处理状态与检索页面 | Closed | L1 + 部分 L2 | 新建 F-015 跟进 501/capability gating 与检索 readiness |
| #111 F-004 Admin 模型 Profile 与解析器配置页面 | Closed | L1 + 部分 L2 | model profile 可继续 L2；parser configs 跟随 F-015 gating |
| #112 F-005 智能问答聊天、SSE、工具摘要与引用展示 | Closed | L1 + 部分 L2 | 新建 F-016，引用和工具展示对齐 #84/#93/QA readiness |
| #113 F-006 QA 配置、检索测试与统计页面 | Closed | L1 + 部分 L2 | 检索测试和统计按 QA/Knowledge readiness 显式降级 |
| #114 F-007 报告模板、素材与报告记录页面 | Closed | L1 + 部分 L2 | 新建 F-014，移除 fallback 并接入真实 Document API |
| #115 F-008 报告大纲、章节编辑、任务进度与导出流程 | Closed | L1 + 部分 L2 | 新建 F-014；AI 生成闭环等待 Document 后续任务 |
| #116 F-009 共享组件、状态覆盖与视觉一致性 | Closed | L1 质量建设 | 不重开；作为后续页面状态验收基线 |
| #117 F-010 关键流程测试与 PR 前检查 | Open，PR #266 已合入 | 测试基础与 API-boundary mock smoke | 不替代真实 L3；真实跨模块 smoke 放入 S-022/#125 |
| #161 F-011 固定 openapi-typescript 并自动生成 Gateway 类型 | Closed | L2 基础设施 | 继续保持 Gateway OpenAPI 为唯一前端类型源 |
| #162 F-012 Gateway 类型落地与 legacy envelope 收敛 | Closed | L2 基础设施 | 后续业务 issue 必须复用 envelope/requestId 规则 |

## 风险规则

- Silent fallback 不得作为验收通过依据。生产路径 API 失败时必须展示错误、未就绪或禁用状态。
- 501 或 NotImplemented 不能被前端吞掉后展示假数据。
- `mockRoutes` 只能用于测试或显式开发场景，不能作为真实业务流程的兜底。
- `job succeeded` 只说明该 job 的当前实现状态成功；对于 Document，非文件类生成 job 不能被解读为真实 AI 正文已经生成。
- 前端不得直连 `services/knowledge`、`services/qa`、`services/document`、`services/file` 或 `ai-gateway` 内部地址。
- 没有 OpenAPI path 的能力不得生成可调用前端 client 方法。

## 后续 Issue 草案

### F-014 Report 前端去除 fallback 并接入真实 Document API

#### 认领规则

- 本任务为自领任务，当前不预分配 Assignee。
- 只允许 1 名主责人完成；认领前请在本 issue 评论 `认领：@你的 GitHub 用户名`，然后将自己设为 Assignee。
- 可以请其他成员 review 或协助排障，但主责人只能有 1 个；如需转让，请在 issue 评论中交接清楚。
- 一切冲突以 `docs/` 为准；如果代码或旧本地草稿与 `docs/` 冲突，按 `docs/` 修改代码或同步公开文档。

#### 任务信息

- 编号：`F-014`
- 状态：`Draft`
- 主责小组：`Frontend`
- View：`Frontend`
- 优先级：`P1`
- 批次：`Batch 4`
- 模块：`frontend`
- Risk：`Normal`
- 依赖任务：#108 #114 #115 #161 #162
- 阻塞任务：无
- 并行任务：#125
- 依赖原因：Document 当前已提供 report types/templates/materials/reports/outlines/sections/jobs/files/settings/statistics/logs 等基础接口，但真实 AI 大纲/正文生成、Document MCP tools、Pandoc/LibreOffice 富 DOCX 和跨服务 smoke 仍未闭环。
- 建议分支：`Frontend/feat/report-real-document-api`
- GitHub Project：`Software Teamwork`
- Project sync：`pending`

#### 权威依据

- `docs/collaboration/frontend-readiness-task-plan.md`
- `docs/architecture/frontend-backend-contract.md`
- `docs/services/gateway/api/openapi.yaml`
- `docs/services/document/docs/implementation.md`
- `.trellis/spec/frontend/quality-guidelines.md`

#### 任务范围

- 移除报告页面中的 `fallbackTypes`、`fallbackTemplates`、`fallbackMaterials`、`fallbackReports`、`fallbackOutline`、`fallbackSections` 等 silent fallback。
- 报告类型、模板、素材、记录、outline、section、job、event、file/content 按 Gateway Document active paths 接入真实 API。
- API 失败时展示错误 envelope 中的 `message` 和 `requestId`；未就绪或依赖失败状态不得用假数据覆盖。
- 对真实 AI 大纲/正文生成、Document MCP tools、富 DOCX 等未实现能力显示“未就绪”或禁用状态。
- 复用已有 typed client、TanStack Query、加载/空/错误/权限状态组件。

#### 交付物

- 报告工作台不再依赖 silent fallback 完成生产路径渲染。
- 已 ready Document API 能真实拉取和提交数据。
- 未 ready 能力有显式状态，不误导 PM 或验收人员。

#### 验收标准

- [ ] 生产路径中不存在报告模块 silent fallback 数据。
- [ ] API 错误会展示可排查的 `requestId`。
- [ ] 501、dependency_error 或未就绪能力不会显示假成功。
- [ ] 基础 CRUD、outline/section 保存、job/event 查询和 report file content 读取按当前 Gateway 契约调用。
- [ ] 不直连 Document/File/AI Gateway 内部地址。

#### 边界与不做内容

- 不实现后端真实 AI 生成、Document MCP tools 或 Pandoc/LibreOffice。
- 不把 API-boundary mock 测试当作真实 L3 smoke。
- 不修改 Gateway OpenAPI，除非管理组另行确认契约变更。

#### PR 要求

- PR 目标分支必须是主仓库 `develop`。
- Commit message 使用 Conventional Commits。
- PR 描述列出完成范围、验证命令、未完成风险和关联 issue。

### F-015 Knowledge 前端按 Gateway 501 状态做 capability gating

#### 认领规则

- 本任务为自领任务，当前不预分配 Assignee。
- 只允许 1 名主责人完成；认领前请在本 issue 评论 `认领：@你的 GitHub 用户名`，然后将自己设为 Assignee。
- 可以请其他成员 review 或协助排障，但主责人只能有 1 个；如需转让，请在 issue 评论中交接清楚。
- 一切冲突以 `docs/` 为准；如果代码或旧本地草稿与 `docs/` 冲突，按 `docs/` 修改代码或同步公开文档。

#### 任务信息

- 编号：`F-015`
- 状态：`Draft`
- 主责小组：`Frontend`
- View：`Frontend`
- 优先级：`P1`
- 批次：`Batch 4`
- 模块：`frontend`
- Risk：`Normal`
- 依赖任务：#84 #85 #110 #161 #162
- 阻塞任务：无
- 并行任务：#125
- 依赖原因：Knowledge 当前已具备知识库 CRUD 和文档上传 handoff，但 chunks/content、knowledge-queries、parser-configs 等 active path 仍存在 NotImplemented 或未完整闭环风险。
- 建议分支：`Frontend/feat/knowledge-capability-gating`
- GitHub Project：`Software Teamwork`
- Project sync：`pending`

#### 权威依据

- `docs/collaboration/frontend-readiness-task-plan.md`
- `docs/architecture/frontend-backend-contract.md`
- `docs/services/gateway/api/openapi.yaml`
- `docs/services/knowledge/docs/implementation.md`
- #84 #85

#### 任务范围

- 保留并校验已 ready 的知识库 CRUD、文档列表、文档上传真实 API 流程。
- 对 document PATCH/DELETE、chunks、content、`knowledge-queries`、parser configs 等未 ready 或可能 501 的能力做 capability gating。
- 页面展示未就绪、依赖失败、权限不足和 requestId，不做静默 fallback。
- 检索页面在 #84 完成前不得展示假检索结果；parser config 管理在 #85 完成前不得假成功。

#### 交付物

- Knowledge 页面能区分“真实可用”“未就绪”“权限不足”“依赖失败”。
- active path 返回 501 或 dependency_error 时，前端不会误判为无数据成功。
- PM 能从页面和文档判断哪些能力等待后端任务。

#### 验收标准

- [ ] 已 ready CRUD/upload 继续走 Gateway 真实 API。
- [ ] chunks/content/knowledge-queries/parser-configs 未 ready 时有禁用或未就绪提示。
- [ ] 不展示 mock 检索结果、不吞掉 501、不把 501 当空列表。
- [ ] 错误提示包含 requestId。
- [ ] 不直连 Knowledge/File/Parser/Qdrant/AI Gateway 内部地址。

#### 边界与不做内容

- 不实现 Knowledge retrieval、Parser runtime、Qdrant 或 embedding/rerank。
- 不替代 #84、#85 或 #125。

#### PR 要求

- PR 目标分支必须是主仓库 `develop`。
- Commit message 使用 Conventional Commits。
- PR 描述列出完成范围、验证命令、未完成风险和关联 issue。

### F-016 QA 聊天引用与工具展示对齐后端能力

#### 认领规则

- 本任务为自领任务，当前不预分配 Assignee。
- 只允许 1 名主责人完成；认领前请在本 issue 评论 `认领：@你的 GitHub 用户名`，然后将自己设为 Assignee。
- 可以请其他成员 review 或协助排障，但主责人只能有 1 个；如需转让，请在 issue 评论中交接清楚。
- 一切冲突以 `docs/` 为准；如果代码或旧本地草稿与 `docs/` 冲突，按 `docs/` 修改代码或同步公开文档。

#### 任务信息

- 编号：`F-016`
- 状态：`Draft`
- 主责小组：`Frontend`
- View：`Frontend`
- 优先级：`P1`
- 批次：`Batch 4`
- 模块：`frontend`
- Risk：`Normal`
- 依赖任务：#84 #93 #112 #113 #161 #162
- 阻塞任务：无
- 并行任务：#125
- 依赖原因：QA sessions/messages/SSE 基础能力已可用于 L2，但真实 RAG 检索依赖 #84，引用快照和引用详情依赖 #93。
- 建议分支：`Frontend/feat/qa-capability-aligned-chat`
- GitHub Project：`Software Teamwork`
- Project sync：`pending`

#### 权威依据

- `docs/collaboration/frontend-readiness-task-plan.md`
- `docs/architecture/frontend-backend-contract.md`
- `docs/services/gateway/api/openapi.yaml`
- `docs/services/qa/docs/implementation.md`
- `docs/services/knowledge/docs/implementation.md`
- #84 #93

#### 任务范围

- 聊天消息和 SSE 展示只使用 QA 返回的安全事件：`message.created`、`agent.iteration.started`、`reasoning.step`、`tool.started`、`tool.completed`、`tool.failed`、`answer.delta`、`citation.delta`、`answer.completed`、`error`。
- 引用角标、引用卡片和引用详情按 #93 readiness 展示；未 ready 时显示未就绪或来源不可用，不造假引用。
- 工具摘要只展示 QA 脱敏后的 tool summary，不展示完整 prompt、私有 chain-of-thought、MCP 原始参数/结果、内部 URL 或 provider 原始错误。
- Knowledge retrieval 未 ready 时展示 RAG 降级态，不展示 mock thinking/tool steps。

#### 交付物

- QA Chat 页面能真实处理 QA SSE 和错误 envelope。
- 引用与工具 UI 和后端能力保持一致。
- 未 ready 能力不会被前端 mock 成已完成。

#### 验收标准

- [ ] 不展示私有 chain-of-thought、完整 prompt、内部 URL、object key 或 provider 原始错误。
- [ ] citation UI 按 #93 readiness 展示；未 ready 时明确说明。
- [ ] Knowledge retrieval 失败或未 ready 时，页面显示降级态和 requestId。
- [ ] tool summary 只使用 QA 返回的脱敏摘要。
- [ ] 不直连 QA/Knowledge/AI Gateway/MCP 内部地址。

#### 边界与不做内容

- 不实现 QA citation 后端、Knowledge retrieval 或 MCP tool runtime。
- 不替代 #84、#93 或 #125。

#### PR 要求

- PR 目标分支必须是主仓库 `develop`。
- Commit message 使用 Conventional Commits。
- PR 描述列出完成范围、验证命令、未完成风险和关联 issue。

### S-022 前端跨模块 E2E smoke 与 issue 验收标准升级

#### 认领规则

- 本任务为自领任务，当前不预分配 Assignee。
- 只允许 1 名主责人完成；认领前请在本 issue 评论 `认领：@你的 GitHub 用户名`，然后将自己设为 Assignee。
- 可以请其他成员 review 或协助排障，但主责人只能有 1 个；如需转让，请在 issue 评论中交接清楚。
- 一切冲突以 `docs/` 为准；如果代码或旧本地草稿与 `docs/` 冲突，按 `docs/` 修改代码或同步公开文档。

#### 任务信息

- 编号：`S-022`
- 状态：`Draft`
- 主责小组：`Special`
- View：`Special`
- 优先级：`P1`
- 批次：`Batch 4`
- 模块：`frontend`
- Risk：`Normal`
- 依赖任务：#117 #125 #163 #264
- 阻塞任务：无
- 并行任务：#84 #85 #93
- 依赖原因：#117/#163 提供前端测试基础和 API-boundary mock 测试，#125 负责跨服务/MCP smoke，本任务负责把前端 issue 验收标准升级为 readiness-aware。
- 建议分支：`Special/test/frontend-readiness-smoke`
- GitHub Project：`Software Teamwork`
- Project sync：`pending`

#### 权威依据

- `docs/collaboration/frontend-readiness-task-plan.md`
- `docs/collaboration/frontend-workflow.md`
- `.trellis/spec/frontend/quality-guidelines.md`
- `docs/architecture/current-capability-matrix.md`
- `docs/architecture/frontend-backend-contract.md`
- #117 #125 #163

#### 任务范围

- 定义前端 issue 的 L1/L2/L3 验收模板，禁止 silent fallback 通过验收。
- 为登录、管理页、QA 聊天、报告工作台制定 readiness-aware smoke 清单。
- 将真实跨模块 smoke 明确路由到 #125，前端侧只记录用户路径、期望状态、skip/expected failure 规则和 requestId 采集。
- 保留 #117/#163 的 mock 测试价值，但在验收文档中说明它们不等于真实后端业务闭环。

#### 交付物

- 前端 issue 创建和验收时可引用的 L1/L2/L3 标准。
- 一份可执行或可人工复核的前端 readiness smoke checklist。
- 现有 closed F issue 和后续 L2/L3 follow-up 的边界清晰。

#### 验收标准

- [ ] checklist 覆盖 login、admin runtime config、Knowledge、QA Chat、Report workspace。
- [ ] 所有 smoke 请求都通过 Gateway `/api/v1/**`。
- [ ] 未 ready 能力只能标记 skipped 或 expected failure，不能假成功。
- [ ] 失败输出必须包含 requestId 或说明无法取得 requestId 的原因。
- [ ] 文档明确 #125 负责真实跨服务/MCP smoke，#163 负责测试基线，不重复造轮子。

#### 边界与不做内容

- 不实现业务功能，不替代 #125 的跨服务脚本。
- 不重开 F-001 到 F-012。
- 不要求未 ready 后端能力在 smoke 中强行通过。

#### PR 要求

- PR 目标分支必须是主仓库 `develop`。
- Commit message 使用 Conventional Commits。
- PR 描述列出完成范围、验证命令、未完成风险和关联 issue。

## 面向 PM 的建议

1. 已关闭的 F-001、F-009、F-011、F-012 主要是前端工程基础、共享组件和类型契约基础，应视为后续 L2/L3 的前置能力，不应解释为业务闭环完成。
2. F-002 可以视为登录和 AppShell 的 L1/L2 基础，建议纳入 S-022 的登录 smoke，而不是新建重复任务。
3. F-003、F-004、F-005、F-006、F-007、F-008 都包含真实页面价值，但需要按模块拆 follow-up：Report 走 F-014，Knowledge 走 F-015，QA Chat 走 F-016。
4. #117 和 #163 保留为前端测试基础，不承担真实后端 readiness 证明；真实跨服务 smoke 归 #125，前端验收升级归 S-022。
5. 创建后续 issue 时，标题和正文应明确目标层级。例如“L2 接入真实 Document API”或“L3 跨模块 smoke”，避免再次把页面骨架和业务闭环混在一起。
6. PM 验收页面演示时，应要求演示者说明每个页面当前处于 L1、L2 还是 L3，并列出未就绪能力的 skip/expected failure 原因。

## 本提案不做的事

- 不直接修改 `apps/web/src`。
- 不重开 F-001 到 F-012。
- 不实现后端 501 path。
- 不替代 #117、#125 或 #163。
- 不创建 GitHub follow-up issue；是否创建由项目负责人审阅本提案后决定。
