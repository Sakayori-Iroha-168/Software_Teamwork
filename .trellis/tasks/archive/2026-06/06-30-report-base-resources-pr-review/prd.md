# Address Report Base Resources PR Review

## Background

PR #254 received Codex PR Review feedback for issue #159 contract alignment.
The feedback points to two concrete gaps:

- `services/document/internal/http/openapi_contract_test.go` currently checks
  broad `$ref` string presence instead of path-level success response schema
  references, and it omits `/report-statistics/overview`.
- `docs/services/document/api/openapi.yaml` documents `GET /report-templates`
  without the implemented `enabled` query parameter.

## Requirements

- Update the OpenAPI contract regression test to parse the YAML structure and
  assert each affected endpoint's successful response schema `$ref` directly.
- Include `/report-statistics/overview` in the path-level schema coverage.
- Add the `enabled` boolean query parameter to `GET /report-templates`.
- Keep the change scoped to document service API contract and regression tests.

## Acceptance Criteria

- The regression test fails if any covered endpoint is wired to an unexpected
  success response schema.
- `GET /report-templates` documents `enabled` consistently with the handler
  and with report material filtering.
- Document service tests and build pass from `services/document`.
- OpenAPI YAML parses successfully.
