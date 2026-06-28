# Conversation History Contract

> Executable backend contract for QA multi-turn context and structured history.

---

## Scenario: QA Conversation History

### 1. Scope / Trigger

Trigger this contract when implementing or changing QA conversation history,
multi-turn context construction, chat stream request shape, message DTOs,
citations, response process steps, or the underlying conversation-history
migrations.

This is a cross-layer contract because the data flows through:

```text
Frontend current message -> HTTP handler -> QA service -> repository/storage
-> QA service DTO aggregation -> HTTP response -> frontend history restore
```

The backend owns persisted history. The frontend must not send previous
messages back to the backend for context reconstruction.

### 2. Signatures

HTTP signatures:

```text
GET  /api/conversations/{conversation_id}
POST /api/chat/stream
```

Temporary authenticated identity header until gateway/auth integration exists:

```text
X-User-ID: <authenticated-user-id>
```

Required database entities:

```text
conversations 1:N messages
messages 1:N message_content_blocks
messages 1:N citations
messages 1:N response_runs
response_runs 1:N response_process_steps
```

The service layer must expose operations equivalent to:

```go
GetHistory(ctx, userID, conversationID string) (ConversationHistory, error)
BuildContext(ctx, userID, conversationID, currentMessage string) (ModelContext, error)
AcceptCurrentMessage(ctx, userID string, request StreamRequest) (StreamAccepted, error)
```

### 3. Contracts

`GET /api/conversations/{conversation_id}` returns a frontend-ready history
payload:

```json
{
  "conversation_id": "conv_1",
  "messages": [
    {
      "id": "msg_1",
      "role": "assistant",
      "status": "failed",
      "sequence_no": 3,
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

History requirements:

- Sort messages by `sequence_no` ascending using stable ordering.
- Include `completed`, `stopped`, and `failed` messages.
- Preserve partial assistant text for stopped or failed responses.
- Include content blocks, process steps under `thinking`, citations, optional
  `error_code`, and timestamps.
- Deny access when the conversation owner does not match the authenticated user.

`POST /api/chat/stream` accepts only the current user message:

```json
{
  "conversation_id": "conv_1",
  "message": "current user message"
}
```

Forbidden request shape:

```json
{
  "conversation_id": "conv_1",
  "message": "current user message",
  "messages": []
}
```

Context construction:

- Load persisted messages by `sequence_no`.
- Append the current user message after persisted history.
- Apply model context bounds before model invocation.
- Do not trust frontend-supplied previous history.

### 4. Validation & Error Matrix

| Condition | Error code | HTTP status |
| --- | --- | --- |
| Missing authenticated user | `unauthorized` | `401` |
| Missing `conversation_id` | `validation_error` | `400` |
| Missing or blank current message | `validation_error` | `400` |
| Unknown fields such as `messages` in stream request | `validation_error` | `400` |
| Multiple JSON objects in one stream request | `validation_error` | `400` |
| Conversation not found | `not_found` | `404` |
| Authenticated user does not own conversation | `forbidden` | `403` |
| Unexpected repository/storage failure | `internal_error` | `500` |

All errors must use the standard backend JSON shape:

```json
{
  "error": {
    "code": "forbidden",
    "message": "conversation access denied",
    "requestId": "req_123"
  }
}
```

### 5. Good/Base/Bad Cases

Good:

- A user sends a follow-up through `POST /api/chat/stream` with only
  `conversation_id` and `message`.
- The backend loads prior messages, builds bounded context, stores the current
  user message, and future requests can use that persisted message.

Base:

- A page refresh calls `GET /api/conversations/{conversation_id}`.
- The response returns text, content blocks, process steps, citations, and
  `failed` or `stopped` status so the UI can restore partial answers.

Bad:

- The frontend sends the entire previous message array to `/api/chat/stream`.
- The backend returns only `completed` messages and drops failed/stopped partial
  answers.
- The backend sorts by creation time instead of `sequence_no`.
- A user can fetch another user's conversation history.

### 6. Tests Required

Service tests:

- History is sorted by `sequence_no`.
- `completed`, `stopped`, and `failed` messages are returned.
- Text content is assembled from text content blocks when message content is
  empty.
- Process steps are preserved and sorted.
- Citations are preserved and sorted.
- Cross-user access returns `forbidden`.
- Model context is bounded and includes the current user message.

HTTP tests:

- `GET /api/conversations/{conversation_id}` returns the structured DTO.
- Cross-user history access returns `403`.
- `POST /api/chat/stream` accepts the current message shape.
- `POST /api/chat/stream` rejects frontend-supplied history.
- `POST /api/chat/stream` rejects multiple JSON objects.

Service-local validation:

```bash
cd services/qa
go test ./...
go build ./cmd/server
```

### 7. Wrong vs Correct

#### Wrong

```json
{
  "conversation_id": "conv_1",
  "message": "follow up",
  "messages": [
    { "role": "user", "content": "old frontend history" }
  ]
}
```

This pushes context ownership to the frontend and can produce stale, tampered,
or oversized model context.

#### Correct

```json
{
  "conversation_id": "conv_1",
  "message": "follow up"
}
```

The backend loads trusted persisted history, sorts it by `sequence_no`, applies
model context bounds, and stores the current message for the next turn.
