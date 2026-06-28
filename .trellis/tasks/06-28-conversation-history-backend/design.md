# Design: Conversation History Backend

## Service Boundary

The feature belongs to `services/qa`. The QA service owns conversation history,
message content blocks, citations, response runs, and process steps. The
gateway/auth integration is represented by an authenticated user ID passed to
the QA service; a full auth service is out of scope.

## Local Service Shape

Create the standard service-local layout:

```text
services/qa/
├── go.mod
├── cmd/server/main.go
├── internal/config/
├── internal/http/
├── internal/service/
├── internal/repository/
├── api/openapi.yaml
├── migrations/
└── README.md
```

## Implementation Strategy

Use only the Go standard library so the service builds without downloading
dependencies. Define repository interfaces and an in-memory repository for
tests/local development. Add SQL migrations documenting the intended PostgreSQL
schema.

The HTTP layer should:

- extract user identity from `X-User-ID` as a temporary gateway-provided
  identity contract,
- parse `GET /api/conversations/{conversation_id}`,
- parse `POST /api/chat/stream`,
- return stable JSON errors.

The service layer should:

- authorize conversation ownership,
- load messages sorted by `sequence_no`,
- aggregate content blocks, citations, and response process steps into
  frontend DTOs,
- include stopped and failed messages,
- build bounded context from prior messages.

## API Contracts

### GET /api/conversations/{conversation_id}

Returns:

```json
{
  "conversation_id": "conv_1",
  "messages": [
    {
      "id": "msg_1",
      "role": "assistant",
      "status": "failed",
      "content": "partial answer",
      "content_blocks": [],
      "thinking": [],
      "citations": [],
      "error_code": "model_timeout",
      "timestamp": "2026-06-28T00:00:00Z"
    }
  ]
}
```

### POST /api/chat/stream

Accepts only:

```json
{
  "conversation_id": "conv_1",
  "message": "current user message"
}
```

The endpoint validates the current message, builds backend context from stored
history, and returns an SSE event describing accepted context metadata. Real
model streaming is out of scope.

## Persistence Contract

Add `0001_create_conversation_history.sql` with tables for conversations,
messages, message content blocks, citations, response runs, and response
process steps. The migration is a contract for future PostgreSQL repository
work even though this task uses an in-memory repository.

## Error Handling

Use the stable error shape:

```json
{
  "error": {
    "code": "forbidden",
    "message": "conversation access denied",
    "requestId": "req_123"
  }
}
```

## Validation

Run from `services/qa`:

```bash
go test ./...
go build ./cmd/server
```
