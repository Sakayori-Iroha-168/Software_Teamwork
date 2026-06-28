# QA Service

智能问答服务，负责 `/api/chat/stream` 的 SSE 输出、意图识别、路由和运行结果持久化。

当前版本先提供最小可运行骨架：

- `intent_status`：输出 `knowledge_qa` 或 `general_chat` 识别结果。
- `thinking_step`：输出后端处理进度。
- `token`：流式输出回答片段。
- `citation`：知识问答路径下输出 mock 引用。
- `done`：标记本轮回答完成。
- `error`：预留异常事件。

## Local Run

```bash
go run ./cmd/server
```

默认监听 `:8080`，可通过 `QA_PORT` 修改。

```bash
curl -N -X POST http://localhost:8080/api/chat/stream \
  -H "Content-Type: application/json" \
  -d "{\"conversation_id\":\"conv_demo\",\"message\":\"帮我检索知识库里的规程\",\"knowledge_bases\":[\"kb1\"]}"
```

## Checks

```bash
go test ./...
go build ./cmd/server
```

## Persistence

数据库与一键部署设计在 `../../docs/qa-system-design/db`：

```bash
cd ../../docs/qa-system-design/db
docker compose up -d
```

当前代码使用内存仓库，字段已按 `qa-system-design/db/init/02_schema.sql` 对齐。接入 PostgreSQL 时替换 `internal/repository` 的实现即可。
