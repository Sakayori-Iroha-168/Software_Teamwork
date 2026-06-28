# PRD: QA Session Resource API

## Goal

Implement the backend API for task DLQA-136 / 1.1:
`POST`, `GET`, `PATCH`, and `DELETE` QA session resource endpoints.

The work must start from the latest `upstream/develop`, not from the old
conversation-history branch. The QA service is implemented in Go under
`services/qa`.

## Source Requirements

- Screenshot: `DLQA-136 1.1 后端：实现 QA 会话资源接口`.
- Uploaded interface document: `D:/download/qa.md`.
- Uploaded database document: `D:/download/qa-database.md`.
- Project backend rules under `.trellis/spec/backend/`.

## API Scope

Implement these service-level routes:

- `POST /api/v1/qa-sessions`
- `GET /api/v1/qa-sessions`
- `GET /api/v1/qa-sessions/{sessionId}`
- `PATCH /api/v1/qa-sessions/{sessionId}`
- `DELETE /api/v1/qa-sessions/{sessionId}`

All routes require `X-User-Id`. `X-Request-Id` must be echoed when present.

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
- List responses aggregate `messageCount` and `lastMessagePreview`.
- Soft-deleted sessions must be hidden from list/detail/update/delete.

## Behavior Requirements

1. `POST /api/v1/qa-sessions`
   - Accepts optional `title`.
   - Creates an `active` session for the current user.
   - Returns `201` with `{data, requestId}`.

2. `GET /api/v1/qa-sessions`
   - Lists only the current user's sessions.
   - Supports `page`, `pageSize`, `status`, and `sort`.
   - Defaults to `page=1`, `pageSize=20`, `status=active`,
     `sort=-updatedAt`.
   - Returns `{data, page, requestId}`.

3. `GET /api/v1/qa-sessions/{sessionId}`
   - Returns only the current user's session.
   - Returns `403` when another user's session is accessed.
   - Returns `404` when the session is missing or deleted.

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
- Unknown session -> `404 not_found`.
- Cross-user access -> `403 forbidden`.

## Acceptance Criteria

- A logged-in user can create and then fetch their own session.
- The list endpoint returns only the current user's sessions and includes
  pagination metadata.
- The list endpoint includes `messageCount` and `lastMessagePreview`.
- Renaming and archiving a session returns the updated camelCase DTO.
- Deleting a session hides it from subsequent list and detail requests.
- Accessing another user's session returns `403` and does not leak data.
- Missing sessions return `404`.
- `go test ./...` passes under `services/qa`.
- `go build -buildvcs=false ./cmd/server` passes under `services/qa`.

## Out Of Scope

- Real PostgreSQL driver integration.
- Gateway route implementation.
- Real auth service integration beyond `X-User-Id`.
- Message creation/SSE generation endpoints from later QA cards.
- QA config, LLM config, citation lookup, retrieval test, and metrics APIs.
