# 处理步骤持久化

> 分支：`LiXiChuanTeam/feat/process-steps-persistence`
> 日期：2026-06-29
> 状态：已实现

## 1. 目标

保存意图、检索、生成、校验等面向用户的处理摘要，支持流式更新和历史恢复。

## 2. 接口

### 写入：`POST /api/v1/qa-sessions/{sessionId}/messages`

通过 SSE `reasoning.step` 事件流式输出处理步骤，回答完成后持久化到 `response_process_steps` 表。

```
event: reasoning.step
id: 3
data: {"type":"tool_call","label":"检索知识库","status":"done","detail":"命中 5 条结果"}
```

### 恢复：`GET /api/v1/qa-sessions/{sessionId}/messages`

通过 `includeThinking` 参数控制是否返回处理步骤，默认 `true`。

```json
{
  "data": [{
    "id": "msg_001",
    "role": "assistant",
    "thinking": [
      {"type": "tool_call", "label": "检索知识库", "status": "done", "detail": "命中 5 条结果"}
    ]
  }]
}
```

### 事件回放：`GET /api/v1/qa-sessions/{sessionId}/events`

按 `responseRunId` 读取短期保存的 SSE 事件，用于断线恢复。

## 3. 数据

| 表 | 用途 |
|----|------|
| `response_process_steps` | 持久化可展示处理步骤摘要 |
| `response_runs` | 关联 step → run → assistant_message |
| `response_stream_events` | SSE 事件短期保存（24h 过期） |

## 4. ThinkingStep 契约

### 允许的 type

| type | 触发时机 |
|------|----------|
| `agent_iteration` | ReAct 迭代开始 |
| `tool_call` | 工具调用开始/完成/失败 |
| `tool_result` | 工具返回结果摘要 |
| `generation` | 模型生成文本 |
| `citation` | 引用确认 |
| `verify` | 校验步骤 |

### 允许的 status

| status | 含义 |
|--------|------|
| `pending` | 步骤尚未开始 |
| `running` | 步骤执行中 |
| `done` | 步骤完成 |
| `failed` | 步骤失败 |

### 安全边界

**不保存**：完整 prompt、MCP 工具参数/结果原文、私有思维链、内部 URL、原始文档全文、向量 payload、provider 原始错误。

## 5. 实现要点

### 写入链路

```
Agent Loop (qa.go Ask)
  → stepFromAgentEvent()            生成 ReasoningStep
  → emit("reasoning.step", ...)     输出 SSE 事件
  → SaveReasoningSteps()           写入 response_process_steps
  → SaveStreamEvents()             写入 response_stream_events
```

### 读取链路

```
GET /messages?includeThinking=true
  → handleListMessages()            解析 includeThinking 参数
  → ListMessagesWithThinking()      查询 messages
  → ListReasoningStepsByMessages()  JOIN response_process_steps + response_runs
  → Message.Thinking               填充 steps 到 JSON 响应
```

### 数据库查询

```sql
SELECT rr.assistant_message_id, rps.step_type, rps.label, rps.detail, rps.status, rps.step_order
FROM response_process_steps rps
JOIN response_runs rr ON rr.id = rps.response_run_id
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.assistant_message_id = ANY($1)   -- 批量查询
  AND c.external_user_id = $2             -- 用户隔离
  AND rr.conversation_id = $3
  AND c.deleted_at IS NULL
ORDER BY rps.step_order
```

## 6. 代码修改清单

| 文件 | 修改 |
|------|------|
| `internal/service/qa.go` | `Message` 加 `Thinking` 字段；`Repository` 接口加 `ListReasoningStepsByMessages`；新增 `ListMessagesWithThinking` |
| `internal/repository/postgres.go` | 新增 `ListReasoningStepsByMessages` 实现 |
| `internal/http/server.go` | `QAService` 接口更新；`handleListMessages` 支持 `includeThinking` |
| `internal/service/qa_test.go` | `fakeRepository` 补全接口 |
| `internal/http/server_test.go` | `fakeQAService` 补全接口 |

## 7. 验收标准

| # | 标准 | 验证 |
|---|------|------|
| 1 | Given Agent Loop 步骤状态变化，When 写入 pending/running/done/failed，Then step_no 唯一且事件与数据库一致 | `(response_run_id, step_order)` UNIQUE 约束 + DELETE-INSERT 原子替换 |
| 2 | Given 回答完成或失败后刷新，When 查询历史消息，Then 终态步骤按顺序恢复且不包含敏感内容 | `ORDER BY step_order` + `MarshalJSON` 只序列化 type/label/status/detail |

## 8. 依赖

- 2.1 — Agent Loop 事件输出（`stepFromAgentEvent` 已经对接）
- Feature 6 repository — `Repository` 接口已扩展
- 1.3 事件输出 — `SaveStreamEvents` 已复用
