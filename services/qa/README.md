# QA Service

Go microservice for intelligent Q&A: streaming chat, thinking-step persistence, and conversation history.

## DLQA-113 scope

- Emit `thinking_step` SSE events from `POST /api/chat/stream`
- Persist safe business step summaries to `response_process_steps`
- Update the same sequence item when a `step_type` transitions `running` → `done`
- Never store or return private chain-of-thought content
- Expose persisted steps via `GET /api/conversations/{id}` as `messages[].thinking`

## Run locally

```bash
export QA_DATABASE_URL="postgres://user:pass@localhost:5432/qa?sslmode=disable"
psql "$QA_DATABASE_URL" -f migrations/0001_create_qa_tables.sql

cd services/qa
go mod tidy
go run ./cmd/server
```

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/conversations` | Create conversation |
| GET | `/api/conversations/{id}` | Conversation detail with thinking steps |
| POST | `/api/chat/stream` | SSE chat stream |

## Tests

```bash
go test ./...
```
