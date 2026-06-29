# QA Service Draft

This service draft implements the QA-owned configuration endpoints from the
canonical skeleton in `docs/services/qa`.

Canonical contract files:

- `docs/services/qa/README.md`
- `docs/services/qa/api/openapi.yaml`
- `docs/services/qa/docs/data-models.md`

Implemented endpoints:

- `GET /api/v1/qa-config-versions/current`
- `POST /api/v1/qa-config-versions`
- `GET /api/v1/llm-config-versions/current`
- `POST /api/v1/llm-config-versions`
- `POST /api/v1/llm-connection-tests`

Scope notes:

- QA config versions are immutable.
- `activate=true` switches the single active version atomically in the current
  in-memory draft repository.
- LLM config stores AI Gateway `profileId`, `modelName`, timeouts, and
  generation parameters only.
- Provider API keys, provider base URLs, and raw provider errors are not
  accepted or returned by this service.
- Retrieval test endpoints are intentionally left to the retrieval-test owner.

Local checks:

```bash
go test ./...
go build ./cmd/server
```

Run locally:

```bash
go run ./cmd/server
```

The service listens on `:8080` by default. Set `QA_PORT` to override it.
