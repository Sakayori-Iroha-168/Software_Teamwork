# QA Service

The QA service owns AI question-answering session state. This implementation
adds the QA session resource API from `docs/services/qa/README.md` and aligns
with the public gateway contract in `docs/services/gateway/api/openapi.yaml`.

## Endpoints

```text
POST   /api/v1/qa-sessions
GET    /api/v1/qa-sessions
GET    /api/v1/qa-sessions/{sessionId}
PATCH  /api/v1/qa-sessions/{sessionId}
DELETE /api/v1/qa-sessions/{sessionId}
```

All business endpoints require gateway context header `X-User-Id`. Responses
use the gateway-style JSON envelope with `requestId`.

`GET /api/v1/qa-sessions` lists only the current user's sessions, supports
`page`, `pageSize`, `status`, `q`, and `sort`, and returns `messageCount` plus
`lastMessagePreview` aggregated from messages. Full message content is exposed
by the messages child resource, not by the session resource.

## Local Development

```bash
go test ./...
go build -buildvcs=false ./cmd/server
go run ./cmd/server
```

The current repository implementation is in-memory so the HTTP contract and
business rules can be tested before PostgreSQL wiring lands.
