# Implementation Plan: Conversation History Backend

## Steps

1. Read backend specs and shared thinking guides.
2. Scaffold `services/qa` as an independent Go module.
3. Add domain models and DTOs for conversations, messages, blocks, citations,
   process steps, and stream requests.
4. Add repository interfaces plus an in-memory repository.
5. Add the conversation service that:
   - checks ownership,
   - sorts by `sequence_no`,
   - aggregates structured history,
   - includes `completed`, `stopped`, and `failed`,
   - builds bounded context.
6. Add HTTP handlers for:
   - `GET /api/conversations/{conversation_id}`,
   - `POST /api/chat/stream`.
7. Add PostgreSQL migration and OpenAPI sketch.
8. Add tests for ordering, partial history, context building, request shape,
   and cross-user denial.
9. Run `go test ./...` and `go build ./cmd/server` in `services/qa`.

## Validation Commands

```bash
cd services/qa
go test ./...
go build ./cmd/server
```

## Risk Notes

- Do not introduce third-party Go dependencies unless necessary.
- Do not implement real LLM calls in this task.
- Do not rely on frontend-supplied prior history.
- Do not leak another user's conversation data.
