# PRD: QA Session Resource API

## Goal

Implement and maintain the backend API for task DLQA-169 / 1.1:
current-user QA session creation, paginated listing, detail, rename/archive, and
soft deletion.

The QA service is implemented in Go under `services/qa`. This branch must stay
based on the latest `upstream/develop`.

## Source Requirements

- Screenshot: `DLQA-169 1.1 Backend: implement QA session resource API`.
- Uploaded interface document: `D:/download/qa.md`.
- Uploaded database document: `D:/download/qa-database.md`.
- Public contract: `docs/services/gateway/api/openapi.yaml`.
- QA behavior notes: `docs/services/qa/README.md`.
- QA data model: `docs/services/qa/docs/data-models.md`.
- Project backend rules under `.trellis/spec/backend/`.

## API Scope

Implement these service-level routes:

- `POST /api/v1/qa-sessions`
- `GET /api/v1/qa-sessions`
- `GET /api/v1/qa-sessions/{sessionId}`
- `PATCH /api/v1/qa-sessions/{sessionId}`
- `DELETE /api/v1/qa-sessions/{sessionId}`

All routes require gateway-injected authenticated user context through
`X-User-Id`. `X-Request-Id` must be echoed when present.

## Data Requirements

- Sessions are backed by `conversations`.
- User isolation is enforced through `external_user_id`.
- Session DTOs are camelCase:
  - `id`
  - `title`
  - `status`
  - `messageCount`
  - `lastMessagePreview`
  - `createdAt`
  - `updatedAt`
- List responses aggregate `messageCount` and `lastMessagePreview` from
  messages.
- Full message content is returned by the messages child resource, not by the
  session resource.
- Soft-deleted sessions must be hidden from list/detail/update/delete.

## Behavior Requirements

1. `POST /api/v1/qa-sessions`
   - Accepts optional `title`.
   - Creates an `active` session for the current user.
   - Returns `201` with `{data, requestId}`.

2. `GET /api/v1/qa-sessions`
   - Lists only the current user's sessions.
   - Supports `page`, `pageSize`, `status`, `q`, and `sort`.
   - Defaults to `page=1`, `pageSize=20`, `status=active`,
     `sort=-updatedAt`.
   - `q` filters within the current user's visible sessions by session title or
     latest message preview.
   - Returns `{data, page, requestId}`.

3. `GET /api/v1/qa-sessions/{sessionId}`
   - Returns only the current user's session.
   - Returns `403` for another user's session and `404` for missing or deleted
     sessions without returning resource data.

4. `PATCH /api/v1/qa-sessions/{sessionId}`
   - Updates `title` and/or `status`.
   - Allows status values `active` and `archived`.
   - Returns the updated `QASession`.

5. `DELETE /api/v1/qa-sessions/{sessionId}`
   - Soft deletes the session.
   - Returns `204` with no response body.

## Error Requirements

Errors must use the stable envelope:

```json
{
  "error": {
    "code": "validation_error",
    "message": "request validation failed",
    "requestId": "req_123",
    "fields": {}
  }
}
```

Expected mappings:

- Missing `X-User-Id` -> `401 unauthorized`.
- Invalid JSON or invalid fields -> `400 validation_error`.
- Unknown or deleted session -> `404 not_found`.
- Cross-user access -> `403 forbidden`.

## Acceptance Criteria

- A logged-in user can create and then fetch their own session.
- The list endpoint returns only the current user's sessions and includes
  pagination metadata.
- The list endpoint includes `messageCount` and `lastMessagePreview`.
- The list endpoint supports `q` filtering without returning another user's
  matching sessions.
- Renaming and archiving a session returns the updated camelCase DTO.
- Deleting a session hides it from subsequent list and detail requests.
- Accessing another user's session returns `403` and does not return session
  data.
- Missing sessions return `404`.
- `go test ./...` passes under `services/qa`.
- `go build -buildvcs=false ./cmd/server` passes under `services/qa`.

## Out Of Scope

- Real PostgreSQL driver integration.
- Gateway route implementation.
- Real auth service integration beyond `X-User-Id`.
- Message creation/SSE generation endpoints from later QA cards.
- QA config, LLM config, citation lookup, retrieval test, and metrics APIs.
