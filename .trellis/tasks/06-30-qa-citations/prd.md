# QA Citation Snapshots and Lookup APIs

## Context

Issue #93 ([B-007] QA citation snapshots, citation detail, and batch lookup) belongs to JerryTeam backend work for the QA service. The current branch is `JerryTeam/feat/qa-citations` and is based on the latest `upstream/develop` at task start.

Authoritative contracts:

- `docs/services/qa/README.md`
- `docs/services/qa/docs/data-models.md`
- `docs/services/gateway/api/openapi.yaml`

Dependencies listed by #93 are already closed: #89, #91, and #84.

## Scope

Implement QA-owned citation snapshot behavior and resource APIs:

- Persist answer-time citation snapshots with stable per-answer `citationNo`.
- Return citations for a message through `GET /api/v1/messages/{messageId}/citations`.
- Return a citation detail snapshot through `GET /api/v1/citations/{citationId}`.
- Return visible citations in bulk through `POST /api/v1/citation-lookups`.
- Keep public responses limited to QA snapshot fields and avoid internal file IDs, object keys, internal URLs, and raw vectors.
- Preserve historical answer citations after the source document changes.
- Degrade cleanly when a source document is missing or no longer visible.

## Out Of Scope

- Returning original full document content. That remains knowledge/file-owned via `/documents/{documentId}/content`.
- Implementing knowledge/file CRUD or visibility rules outside QA-owned resource filtering.
- Frontend rendering of superscripts/cards/source detail.

## Acceptance Criteria

- Authorized users can list citations for their own messages.
- Authorized users can fetch citation detail for their own citations.
- Batch lookup returns only citations visible to the current user and does not reveal omitted citation IDs.
- Unauthorized cross-user access returns standard 404/403-style resource responses without leaking existence.
- Citation responses do not expose internal storage identifiers, object keys, internal URLs, or vector payloads.
- Missing or invisible source documents still return the saved citation snapshot with an unavailable source marker.
- Citation numbers are stable within the same answer.

## Validation Plan

- Run Go formatting on QA service files.
- Run QA service tests.
- Run QA service build with VCS stamping disabled when required by the local worktree.
- Run `git diff --check`.
- Add focused tests for authorization, batch filtering, source-unavailable degradation, and saved citation numbering.
