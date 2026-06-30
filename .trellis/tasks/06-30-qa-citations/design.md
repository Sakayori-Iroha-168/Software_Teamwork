# Design

## Data Model

The QA service owns saved citation snapshots in the `citations` table. The implementation should use existing columns from `docs/services/qa/docs/data-models.md`: citation number, external document/chunk identifiers, display document name, quote text, context, page number, score, rerank score, metadata, chunk type, and source availability state.

The API must expose only public snapshot fields. Repository rows may store external IDs needed for stable references, but responses must not include file storage internals such as object keys, internal URLs, or vectors.

## Service Flow

Citation APIs should follow the same resource ownership pattern as sessions, messages, runs, and SSE:

1. Extract the authenticated external user from gateway/auth context.
2. Resolve resources through repository queries scoped by owner.
3. Return standard `{data, requestId}` envelopes for success.
4. Return standard error envelopes for unauthorized, missing, or invalid requests.

For answer generation, persist citations when the agent/tool pipeline confirms references. Each assistant answer should assign `citationNo` from the order of saved references for that message. Repeated source records within the same answer must keep deterministic ordering.

## Source Availability

Public citation detail should keep the saved quote/context available even when the upstream source document is missing or not visible. In that case:

- `isSourceAvailable` is false.
- `source.available` is false.
- `source.reason` contains a stable non-sensitive reason.
- No internal URL or storage key is returned.

## Gateway Contract

Gateway active paths remain the public `/api/v1/...` routes. QA service handlers remain internal `/internal/v1/...` routes behind the gateway proxy.

## Tests

Tests should cover repository and HTTP/service behavior where possible:

- Owner filtering for list/detail/batch lookup.
- Batch lookup silently omits invisible citations.
- Missing or invisible source degradation.
- Citation snapshot output shape avoids sensitive fields.
- Citation numbering stays stable for saved generated citations.
