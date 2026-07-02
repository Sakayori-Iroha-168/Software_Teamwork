# Frontend Unit Test Strategy Evidence

## 测试范围

- Issue：`T-009` / 待回填 GitHub issue 编号
- 被测分支：`Test/docs/frontend-unit-test-strategy`
- 被测 commit：`f8a4c1e`
- Base branch：`origin/develop @ f70652e`
- PR head：`f8a4c1e`
- 测试负责人：待回填 GitHub 用户名
- 测试环境：本地 Windows PowerShell；Node `v24.11.1`；npm `11.6.2`
- 测试层级：本地自动化；前端静态检查；前端单元测试

## 已运行命令与结果

| 命令或操作                                              | 结果 | 证据                                                                                                                                                                                                                                             |
| ------------------------------------------------------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `git pull --ff-only origin develop`                     | pass | 已从 `8d226de` fast-forward 到 `f70652e`。                                                                                                                                                                                                       |
| `git checkout -B Test/docs/frontend-unit-test-strategy` | pass | 已基于最新 `develop` 创建/重置任务分支，当前 PR head 为 `f8a4c1e`。                                                                                                                                                                              |
| `npm.cmd run typecheck`                                 | pass | `tsc --noEmit` 通过。                                                                                                                                                                                                                            |
| `npm.cmd run typecheck:test`                            | pass | `tsc -p tsconfig.test.json --noEmit` 通过。                                                                                                                                                                                                      |
| `npm.cmd run lint`                                      | pass | `eslint .` 通过。                                                                                                                                                                                                                                |
| `npm.cmd run format:check`                              | fail | Prettier 报告 38 个已有文件格式不一致，包括 `components.json`、`eslint.config.js`、`src/api/citations.ts`、`src/components/ui/button.tsx`、`src/pages/admin/qa-settings.tsx`、`tsconfig.json` 等。未在本测试任务中批量格式化，避免扩大修改范围。 |
| `npm.cmd run build`                                     | pass | 用户普通 PowerShell 复跑通过：716 modules transformed，产物包括 `dist/index.html`、`dist/assets/index-D0u1oJj0.css`、`dist/assets/index-BIkzWWtZ.js`；存在 chunk size warning，但构建成功。                                                      |
| `npm.cmd run test:unit`                                 | pass | 用户普通 PowerShell 复跑通过：19 test files passed，66 tests passed。`src/pages/knowledge/documents/page.test.tsx` 有 React `act(...)` warning，但未导致测试失败。                                                                               |
| `git diff --check`                                      | pass | 当前改动无 whitespace error。                                                                                                                                                                                                                    |

## 未运行项

| 测试项                     | 未运行原因                                                       | 缺失环境                                                  | 残余风险                                           | 后续归属                             |
| -------------------------- | ---------------------------------------------------------------- | --------------------------------------------------------- | -------------------------------------------------- | ------------------------------------ |
| `npm.cmd run dev` 页面冒烟 | 本任务聚焦测试策略、静态检查和单元测试证据，未要求人工页面验收。 | 浏览器人工检查环境。                                      | 无法在本记录中证明页面实际可打开或无白屏。         | 测试负责人在需要人工验收时补充截图。 |
| `npm.cmd run test:e2e`     | 本任务不做完整 E2E，且无真实后端或稳定 mock E2E 证据要求。       | Playwright 浏览器环境；必要时还需要 Gateway/Auth 等后端。 | 不覆盖登录、Knowledge、QA、Report 的端到端页面流。 | 单独 E2E 测试任务。                  |
| 真实后端联调               | 任务边界声明不接入真实后端服务。                                 | Gateway/Auth/Knowledge/QA/Document/File/Parser 等服务。   | 不证明真实接口、鉴权、SSE 或跨服务链路。           | 对应 owner 测试任务。                |

## 缺陷处理

| 问题                                                                                       | 等级             | 处理结论                                                                                 | 关联 issue / PR                           | 复现或验证                                                                 |
| ------------------------------------------------------------------------------------------ | ---------------- | ---------------------------------------------------------------------------------------- | ----------------------------------------- | -------------------------------------------------------------------------- |
| `format:check` 发现 38 个已有文件格式不一致。                                              | 小问题或治理问题 | 本任务先记录，不批量格式化无关文件；建议新建或关联格式清理 issue 后回填编号。            | 待回填格式清理 issue。                    | `cd apps/web && npm.cmd run format:check`                                  |
| Vite/Vitest 在 Codex Windows 终端出现 Tailwind/Rolldown 原生依赖加载失败和 `spawn EPERM`。 | 环境阻塞         | 用户普通 PowerShell 复跑 `build` 和 `test:unit` 均通过，确认不是项目构建或单元测试失败。 | 无需新建缺陷；记录为 Codex 终端环境差异。 | `cd apps/web && npm.cmd run build`；`cd apps/web && npm.cmd run test:unit` |

## 证据清单

- 测试策略文档：`docs/testing/frontend-unit-test-strategy.md`
- 轻量执行记录：`docs/testing/reports/2026-07-02/frontend-unit-test-strategy-evidence.md`
- 测试报告：不适用；本轮为纯前端静态检查和单元测试策略记录，按当前规则使用轻量证据记录。
- 日志：见本文件“已运行命令与结果”。

## 最终结论

测试失败已记录，尚未转 issue：前端 `typecheck`、`typecheck:test`、`lint`、`build` 和 `test:unit` 均已通过；`format:check` 发现当前 `develop` 上 38 个已有前端文件不符合 Prettier 格式规则，本任务暂不扩大范围批量格式化，建议新建或关联格式清理 issue 后回填编号。Codex Windows 终端曾出现 Vite/Tailwind/Rolldown 原生依赖 `spawn EPERM`，但用户普通 PowerShell 复跑 `build` 和 `test:unit` 已通过，记录为环境差异。
