# PRD: Conversation History Backend

## Goal

Implement the backend support for task 3.3.2: maintain multi-turn context on
the backend and return structured conversation history for the frontend.

The backend must let the frontend send only the current chat message while the
QA service loads prior messages by `sequence_no`, builds a bounded model
context, and returns a complete structured history that includes partial
answers from `stopped` or `failed` runs.

## Background

Confirmed facts from the repository and task screenshot:

- Backend implementation must be written in Go.
- Backend services live under `services/<service>/` as independent Go modules.
- The relevant service is `services/qa` because the feature owns AI question
  answering and chat history.
- Existing `services/qa` contains only `.gitkeep`, so this task must scaffold
  the QA service structure needed for the feature.
- The target branch is `JerryTeam/feat/conversation-history`, created from
  latest `upstream/develop`.
- The frontend story is 3.3.1; this backend story is responsible only for
  backend/API/database adaptation.

## Requirements

1. Add a Go QA service under `services/qa`.
2. Model the conversation history ER shape:
   - `conversations` 1:N `messages`,
   - `messages` 1:N `message_content_blocks`,
   - `messages` 1:N `citations`,
   - `response_runs` 1:N `response_process_steps`.
3. Add a migration that captures the schema needed for structured history.
4. Implement `GET /api/conversations/{conversation_id}`.
5. The history response must aggregate messages into a frontend `Message` DTO
   containing:
   - `status`,
   - `content`,
   - `content_blocks`,
   - `thinking`,
   - `citations`,
   - optional `error_code`,
   - `timestamp`.
6. History must be sorted by `sequence_no` in stable ascending order.
7. History must include partial assistant answers for `stopped` and `failed`
   messages. It must not return only `completed` messages.
8. Cross-user conversation access must be denied.
9. Backend context building must read current conversation messages by
   `sequence_no` and produce a bounded context for model calls.
10. `POST /api/chat/stream` must accept only the current message from the
    frontend. The frontend must not send previous history.
11. Handler responses must use stable JSON error responses.
12. Add tests for ordering, partial-history recovery, and cross-user denial.

## Acceptance Criteria

- Consecutive follow-up questions can use the same conversation ID and backend
  context is built from persisted prior messages.
- Refreshing the frontend can recover message text, content blocks, processing
  steps, citations, and failed/stopped statuses.
- The history endpoint returns messages sorted by `sequence_no`.
- The history endpoint includes `completed`, `stopped`, and `failed` messages.
- A user cannot read another user's conversation.
- `POST /api/chat/stream` request shape contains the current message only.
- `go test ./...` passes under `services/qa`.
- `go build ./cmd/server` passes under `services/qa`.

## Out Of Scope

- Real LLM integration.
- Real PostgreSQL driver integration.
- Gateway or auth service implementation.
- Frontend implementation.
- RAG retrieval implementation.

## Open Questions

None. The task screenshot and repository specs define the required scope.
