# QA Service

The QA service owns AI question-answering session state. This implementation
adds the QA session resource API from `docs/services/qa.md`.

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

## Local Development

```bash
go test ./...
go build -buildvcs=false ./cmd/server
go run ./cmd/server
```

The current repository implementation is in-memory so the HTTP contract and
business rules can be tested before PostgreSQL wiring lands.
