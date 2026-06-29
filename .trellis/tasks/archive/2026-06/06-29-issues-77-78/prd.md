# Implement Auth Issues 77 and 78

## Goal

Implement the Batch 1 auth work described by GitHub issues #77 and #78: user creation, session creation, opaque bearer token handling, RBAC identity payloads, current-user/session lookup support, session revocation, and security-event persistence. The implementation must follow the public gateway schemas and auth service contracts under `docs/`.

## Requirements

* Implement auth internal service endpoints for user creation, session creation, session lookup, permission lookup, and session revocation under `/internal/v1/**`.
* Align successful create-user and create-session responses with gateway `SessionResponse`: `data.user`, `data.session`, and `requestId`.
* Store passwords as `argon2id-v1` PHC strings using the documented parameters.
* Issue non-JWT opaque bearer access tokens and persist only versioned `hmac-sha256` token hashes.
* Return gateway-cacheable identity payloads containing `UserSummary`, `SessionSummary`, roles, and permissions.
* Provide source data needed by gateway `/api/v1/users/me` via auth user/session/permission reads.
* Revoke sessions through auth source data so a revoked token/session is no longer active.
* Seed or otherwise ensure baseline roles and permissions exist for standard user, admin, and super admin.
* Persist security events for user creation, session creation, login failure, and session revocation.
* Keep auth scoped to identity/RBAC only; do not implement knowledge, file, report, QA, or domain-resource authorization in auth.

## Acceptance Criteria

* [ ] `POST /internal/v1/users` creates a user, credential, initial role assignment, session, and security events.
* [ ] `POST /internal/v1/sessions` authenticates a user and returns `SessionResponse` data without leaking account enumeration details on failure.
* [ ] Password hashes and token hashes are persisted; plaintext passwords/tokens are not persisted or logged.
* [ ] Token hash generation is covered by tests.
* [ ] Duplicate username, wrong password, missing/invalid auth context, and session revocation paths are covered by tests.
* [ ] `GET /internal/v1/sessions/{sessionId}`, `GET /internal/v1/users/{userId}`, and `GET /internal/v1/users/{userId}/permissions` provide data needed for gateway identity/cache repair.
* [ ] Revoked sessions are no longer returned as active token identities.
* [ ] Unauthorized and forbidden cases are mapped distinctly where applicable.
* [ ] Service-local auth checks pass: `go test ./...` and `go build ./cmd/server`.

## Definition of Done

* Auth service code and tests updated.
* Gateway/public schema compatibility checked against `docs/services/gateway/api/openapi.yaml`.
* Sensitive data remains out of responses, logs, and test assertions except deliberate non-secret hash checks.
* Relevant docs updated only if implementation exposes a contract difference.

## Technical Approach

Implement the missing auth behavior in the existing `services/auth` Go module rather than creating a new service. Reuse the current repository/sqlc/migration baseline, add service-layer primitives for password hashing, token generation/hash, session response assembly, and security events, then expose them through `internal/http/server.go`.

Gateway currently has only skeleton health routes, so this task focuses on auth service source-of-truth capabilities and response payloads. Gateway Redis cache integration remains a downstream gateway implementation task unless already present locally.

## Decision (ADR-lite)

**Context**: Issues #77 and #78 depend on the existing auth service scaffold and docs-driven contracts.

**Decision**: Implement auth source-of-truth behavior inside `services/auth`, aligned with gateway schemas but exposed through internal `/internal/v1/**` routes.

**Consequences**: This keeps auth independently testable and avoids putting identity logic in gateway. Gateway still needs a separate route/proxy/cache integration slice to expose public `/api/v1/**` behavior end-to-end.

## Out of Scope

* Gateway Redis session-cache implementation unless existing gateway code already has the necessary auth routes.
* Public frontend `/api/v1/**` auth handlers in gateway.
* Knowledge, document, file, QA, or resource-level ACL logic.
* Refresh tokens, JWTs, organization-level permissions, and audit-query UI.

## Technical Notes

* Issue #77: <https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/77>
* Issue #78: <https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/78>
* Primary docs: `docs/services/auth/README.md`, `docs/services/auth/docs/data-models.md`, `docs/services/gateway/api/openapi.yaml`, `docs/architecture/frontend-backend-contract.md`.
* Existing code: `services/auth` has a Go module, migration, sqlc queries, repository adapter, and health/readiness handlers; registration/login/logout are currently out of scope in the scaffold README and must be implemented now.
