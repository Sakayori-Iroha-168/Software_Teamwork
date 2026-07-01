# Local Integration Environment

This directory is the S-05 local/demo integration baseline. It starts shared
infrastructure plus the backend service loop through gateway. It is not a
production deployment baseline.

## Entry Points

- Browser/frontend entrypoint: `http://localhost:8080` through gateway only.
- Do not point frontend code at `auth`, `file`, `knowledge`, `qa`,
  `document`, `ai-gateway`, PostgreSQL, Redis, Qdrant, or MinIO directly.
- Internal service ports are exposed for local debugging only.

## Start

```powershell
cd deploy
Copy-Item .env.example .env
docker compose up -d --build
```

Optional AI Gateway:

```powershell
cd deploy
docker compose --profile ai up -d --build
```

Default seeded login:

```text
username: admin
password: LocalDemoAdmin#12345
```

These credentials and all secrets in `.env.example` are local placeholders.
Replace them for any shared or long-lived environment.

## Docker Images Required

If Docker has no local images, install them with:

```powershell
docker pull postgres:16-alpine
docker pull redis:7-alpine
docker pull minio/minio:RELEASE.2025-09-07T16-13-09Z
docker pull minio/mc:RELEASE.2025-08-13T08-35-41Z
docker pull golang:1.25-alpine
docker pull alpine:3.22
```

Then build service images:

```powershell
cd deploy
docker compose build
docker compose --profile ai build
```

The local MinIO server, MinIO `mc`, Redis, PostgreSQL, and Alpine runtime
images are pinned to explicit tags in this repository.

## Ports

| Component | Host port | Container port | Purpose |
| --- | ---: | ---: | --- |
| gateway | 8080 | 8080 | Browser/backend entrypoint |
| auth | 8001 | 8001 | Internal auth service |
| file | 8082 | 8082 | Internal file service |
| knowledge | 8083 | 8083 | Vendor contract adapter |
| qa | 8084 | 8084 | Internal QA service |
| document | 8085 | 8085 | Internal document service |
| ai-gateway | 8086 | 8086 | Optional model/profile service |
| postgres | 5432 | 5432 | Local relational databases |
| redis | 6379 | 6379 | Sessions, queues, coordination |
| minio | 9000/9001 | 9000/9001 | Object storage and console |

Override host ports in `deploy/.env`.

## Environment Variables

| Variable | Service | Required | Description |
| --- | --- | --- | --- |
| `INTERNAL_SERVICE_TOKEN` | gateway/auth/knowledge/qa/ai-gateway | yes | Local service-to-service token placeholder. |
| `TOKEN_HASH_SECRET` | gateway/auth | yes | Local HMAC secret for opaque token hashes. |
| `GATEWAY_AUTH_BASE_URL` | gateway | set in Compose | Internal auth base URL. |
| `GATEWAY_KNOWLEDGE_BASE_URL` | gateway | set in Compose | Internal knowledge base URL. |
| `GATEWAY_QA_BASE_URL` | gateway | set in Compose | Internal QA base URL. |
| `GATEWAY_DOCUMENT_BASE_URL` | gateway | set in Compose | Internal document base URL. |
| `GATEWAY_AI_GATEWAY_BASE_URL` | gateway | set in Compose | Internal AI Gateway base URL; route calls require optional profile to run. |
| `AUTH_DATABASE_URL` | auth | yes | Auth PostgreSQL DSN. |
| `FILE_DATABASE_URL` | file | yes | File metadata PostgreSQL DSN. |
| `FILE_STORAGE_BACKEND` | file | no | `local` in Compose for durable local smoke tests. |
| `DATABASE_URL` | knowledge | no | Optional PostgreSQL for parser-config admin routes. |
| `VENDOR_RUNTIME_URL` | knowledge | yes | RAGFlow vendor HTTP base URL (default in Compose: `http://knowledge-runtime-api:9380`). |
| `KNOWLEDGE_AUTO_START_INGESTION` | knowledge | no | Auto-call vendor `/documents/parse` after upload (default `true`). |
| `QA_DATABASE_URL` | qa | yes | QA PostgreSQL DSN. |
| `KNOWLEDGE_SERVICE_URL` | qa | yes | Internal Knowledge Service URL. |
| `AI_GATEWAY_URL` | qa | yes | Internal chat completions URL; useful when `--profile ai` is running. |
| `DOCUMENT_DATABASE_URL` | document | yes | Document PostgreSQL DSN. |
| `DOCUMENT_REDIS_ADDR` | document | yes | Redis/asynq endpoint. |
| `DOCUMENT_FILE_SERVICE_URL` | document | yes | Internal File Service URL. |
| `DOCUMENT_FILE_SERVICE_TOKEN` | document | yes | Local service token for File Service calls without gateway request context. |
| `DOCUMENT_AI_GATEWAY_URL` | document | yes | Internal AI Gateway base URL. |
| `DOCUMENT_AI_GATEWAY_PROFILE_ID` | document | yes | Seeded placeholder profile id, `default-chat`. |
| `DOCUMENT_AI_GATEWAY_SERVICE_TOKEN` | document | yes | Local service token for AI Gateway internal profile APIs. |
| `AI_GATEWAY_DATABASE_URL` | ai-gateway | yes | AI Gateway PostgreSQL DSN. |
| `AI_GATEWAY_SERVICE_TOKEN_HASHES` | ai-gateway | yes | SHA-256 hashes for allowed service tokens. |
| `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF` | ai-gateway | yes | Local encryption key reference placeholder. |
| `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY` | ai-gateway | yes | Local encryption key placeholder. |

## Health And Readiness

Use gateway for the top-level signal:

```powershell
Invoke-RestMethod http://localhost:8080/healthz
Invoke-RestMethod http://localhost:8080/readyz
```

Service-level readiness endpoints:

```powershell
Invoke-RestMethod http://localhost:8001/readyz
Invoke-RestMethod http://localhost:8082/readyz
Invoke-RestMethod http://localhost:8083/readyz
Invoke-RestMethod http://localhost:8084/readyz
Invoke-RestMethod http://localhost:8085/readyz
Invoke-RestMethod http://localhost:8086/readyz
Invoke-RestMethod http://localhost:8087/readyz
```

`gateway /readyz` checks Redis and auth, and verifies owner service URLs are
configured. Auth, document, and ai-gateway readiness identify PostgreSQL
problems. Compose health checks identify container-level dependency failures.

## Seed Data

`seed-local` applies `deploy/seeds/001-local-demo-seed.sql` after migrations:

- admin user `admin` with local password `LocalDemoAdmin#12345`;
- admin role assignment;
- report types `summer_peak_inspection` and `coal_inventory_audit`;
- knowledge base `kb_local_demo`;
- optional AI model profile placeholders `default-chat`, `default-embedding`,
  and `default-rerank`.

The AI profiles are enabled local placeholders for readiness checks and include
fake encrypted provider credentials. They are not real API keys, so model
invocation still requires operators to configure a real provider key.
Their default provider URL is `http://host.docker.internal:11434/v1`; Compose
maps that hostname to the Docker host for Linux engines with
`host.docker.internal:host-gateway`.

## Request Id Troubleshooting

Every service returns or propagates `X-Request-Id`.

```powershell
$rid = "req_local_debug_001"
Invoke-RestMethod http://localhost:8080/readyz -Headers @{ "X-Request-Id" = $rid }
docker compose logs gateway auth knowledge qa document | Select-String $rid
```

For frontend issues, capture the response `requestId` or `X-Request-Id`, then
search gateway logs first. If gateway reports a dependency error, search the
same id in the owner service logs.

## Knowledge Integration Notes

Knowledge active operations are exposed through gateway:

```powershell
# after logging in and setting $token to the returned access token
$headers = @{ Authorization = "Bearer $token"; "X-Request-Id" = "req_knowledge_local_001" }
Invoke-RestMethod "http://localhost:8080/api/v1/knowledge-bases" -Headers $headers
Invoke-RestMethod "http://localhost:8080/api/v1/knowledge-bases/kb_local_demo/documents" -Headers $headers
Invoke-RestMethod "http://localhost:8080/api/v1/documents/<documentId>/chunks" -Headers $headers
Invoke-WebRequest "http://localhost:8080/api/v1/documents/<documentId>/content" -Headers $headers -OutFile .\knowledge-content.bin
Invoke-RestMethod "http://localhost:8080/api/v1/knowledge-queries" -Method Post -Headers $headers -ContentType "application/json" -Body '{"query":"local demo","topK":3}'
```

Knowledge routes require the RAGFlow runtime at `VENDOR_RUNTIME_URL`
(default `http://knowledge-runtime-api:9380` inside Compose). With the
`knowledge-v2` profile you can run `knowledge-runtime-api` and
`knowledge-runtime-worker` in compose:

```powershell
cd deploy
docker compose --profile knowledge-v2 up -d elasticsearch knowledge-minio-init knowledge-runtime-api knowledge-runtime-worker knowledge
```

Use an explicit `VENDOR_RUNTIME_URL` override only when the runtime is started
outside Compose for local Python development.

See `services/knowledge/runtime/README.md` and `services/knowledge-runtime/README.md`.

## Common Dependency Failures

| Symptom | Likely cause | Check |
| --- | --- | --- |
| `gateway /readyz` returns `502 dependency_error` | Redis or auth is not ready | `docker compose ps`, `docker compose logs redis auth gateway` |
| `auth /readyz` returns `postgres unavailable` | Auth migration or PostgreSQL failed | `docker compose logs postgres migrate-auth auth` |
| Knowledge upload/query returns `502 dependency_error` | Vendor runtime unreachable or ES/MinIO not ready | `docker compose logs knowledge`, verify vendor :9380 and `knowledge-v2` profile |
| Document readyz returns dependency error | Document DB migration failed or DB is unreachable | `docker compose logs migrate-document document postgres` |
| QA message call fails on model invocation | Optional `ai-gateway` profile not running, fake local credential still in use, or host provider is not listening on `host.docker.internal:11434` | `docker compose --profile ai ps`, `docker compose logs ai-gateway qa` |
| MinIO bucket missing | `minio-init` did not complete | `docker compose logs minio minio-init` |
| Host port conflict | Another local process uses a default port | Change the matching `*_PORT` in `deploy/.env` |

## Reset

```powershell
cd deploy
docker compose down -v
docker compose --profile ai down -v
```
