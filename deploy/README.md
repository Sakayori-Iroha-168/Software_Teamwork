# Local Integration Environment

This directory provides the local/demo baseline for issue #122 / S-05. It is not a production deployment.

## Start

```bash
cp deploy/.env.example deploy/.env
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --build
```

The public backend entrypoint is the gateway:

```text
http://localhost:8080
```

Frontend and external callers should use gateway `/api/v1/**` routes only. Do not configure browser code to call `auth`, `file`, `knowledge`, `qa`, `document`, `ai-gateway`, Postgres, Redis, Qdrant, or MinIO directly.

## Seed Data

`seed-demo` inserts reproducible local/demo records after migrations finish:

| Area | Seed |
| --- | --- |
| Auth | `admin` user with `super_admin` role |
| AI Gateway | `default-chat`, `default-embedding`, and `default-rerank` placeholder profiles without provider credentials |
| Document | Demo report types |
| Knowledge | `kb_local_demo` example knowledge base |

Demo login:

```text
username: admin
password: admin-local-password
```

These values are examples only. They must not be reused in production or committed as real secrets.

## Ports

| Service | Host Port | Container Port | Purpose |
| --- | ---: | ---: | --- |
| gateway | `8080` | `8080` | Public backend entrypoint |
| auth | `8001` | `8001` | Internal auth service |
| file | `8082` | `8082` | Internal file service |
| knowledge | `8083` | `8083` | Internal knowledge service |
| qa | `8084` | `8084` | Internal QA service |
| document | `8085` | `8085` | Internal document service |
| ai-gateway | `8086` | `8086` | Internal model profile and invocation service |
| postgres | `5432` | `5432` | Shared local Postgres with one database per service |
| redis | `6379` | `6379` | Shared local Redis |
| qdrant | `6333`, `6334` | `6333`, `6334` | Vector store |
| minio | `9000`, `9001` | `9000`, `9001` | Object store API and console |

Override host ports in `deploy/.env` when a local port is already occupied.

## Environment Variables

Shared:

| Variable | Description |
| --- | --- |
| `INTERNAL_SERVICE_TOKEN` | Local service-to-service token forwarded as `X-Service-Token`. Example only. |
| `INTERNAL_SERVICE_TOKEN_SHA256` | SHA-256 hash of `INTERNAL_SERVICE_TOKEN`, used by AI Gateway. |
| `POSTGRES_PASSWORD` | Local superuser password for the shared Postgres container. Example only. |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Local MinIO credentials. Example only. |

Service-specific variables are kept aligned with each service's `internal/config` package:

| Service | Variables |
| --- | --- |
| gateway | `GATEWAY_HTTP_ADDR`, `GATEWAY_AUTH_BASE_URL`, `GATEWAY_KNOWLEDGE_BASE_URL`, `GATEWAY_QA_BASE_URL`, `GATEWAY_DOCUMENT_BASE_URL`, `GATEWAY_AI_GATEWAY_BASE_URL`, `GATEWAY_REDIS_ADDR`, `GATEWAY_TOKEN_HASH_SECRET`, `GATEWAY_INTERNAL_SERVICE_TOKEN`, `GATEWAY_CORS_ALLOWED_ORIGINS` |
| auth | `AUTH_HTTP_ADDR`, `AUTH_DATABASE_URL`, `AUTH_INTERNAL_SERVICE_TOKEN`, `AUTH_TOKEN_HASH_SECRET`, `AUTH_SESSION_TTL` |
| file | `FILE_HTTP_ADDR`, `FILE_STORAGE_BACKEND`, `FILE_LOCAL_STORAGE_DIR` |
| knowledge | `KNOWLEDGE_HTTP_ADDR`, `DATABASE_URL`, `FILE_SERVICE_BASE_URL`, `KNOWLEDGE_REDIS_ADDR`, `KNOWLEDGE_SERVICE_TOKEN` |
| qa | `QA_HTTP_ADDR`, `QA_DATABASE_URL`, `KNOWLEDGE_SERVICE_URL`, `AI_GATEWAY_URL`, `AI_GATEWAY_TOKEN`, `AI_GATEWAY_TOKEN_HEADER`, `MODEL_ID`, `INTERNAL_SERVICE_TOKEN`, `MCP_TRANSPORT` |
| document | `DOCUMENT_HTTP_ADDR`, `DOCUMENT_DATABASE_URL`, `DOCUMENT_REDIS_ADDR`, `DOCUMENT_FILE_SERVICE_URL`, `DOCUMENT_AI_GATEWAY_URL`, `DOCUMENT_AI_GATEWAY_PROFILE_ID` |
| ai-gateway | `AI_GATEWAY_HTTP_ADDR`, `AI_GATEWAY_DATABASE_URL`, `AI_GATEWAY_SERVICE_TOKEN_HASHES`, `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF`, `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY` |

## Health And Readiness

Check service readiness through gateway first:

```bash
curl -i http://localhost:8080/readyz
```

Then inspect individual services when gateway is not ready:

```bash
curl -i http://localhost:8001/readyz  # auth
curl -i http://localhost:8082/readyz  # file
curl -i http://localhost:8083/readyz  # knowledge
curl -i http://localhost:8084/readyz  # qa
curl -i http://localhost:8085/readyz  # document
curl -i http://localhost:8086/readyz  # ai-gateway
```

Compose uses `ai-gateway /healthz` for container health because the seeded
profiles are placeholders without provider credentials. `ai-gateway /readyz`
is still the diagnostic endpoint for real model readiness and will report a
degraded state until chat, embedding, and rerank profiles are configured with
usable credentials.

Use Compose health state to locate dependency failures:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml ps
docker compose --env-file deploy/.env -f deploy/docker-compose.yml logs gateway auth postgres redis
```

Request-id troubleshooting:

1. Capture `X-Request-Id` from the gateway response header or JSON envelope.
2. Search gateway logs for the request id.
3. Follow downstream logs using the same request id. Gateway forwards it to domain services when proxying.
4. If the response is `dependency_error`, inspect the named service and infrastructure health in `docker compose ps`.

Common failures:

| Symptom | Likely Cause | Check |
| --- | --- | --- |
| `gateway /readyz` reports dependency error | Redis or auth is not ready, or a service URL is missing | `docker compose ps redis auth gateway` |
| Auth is not ready | `auth_system` migration failed or Postgres is unavailable | `docker compose logs migrate-auth auth postgres` |
| AI Gateway container is unhealthy | service process failed or credential encryption env is invalid | `docker compose logs ai-gateway` |
| AI Gateway `/readyz` is degraded | local placeholder profiles have no provider credentials yet | configure real chat, embedding, and rerank profiles through the service API |
| QA exits on startup | `AI_GATEWAY_URL` or `KNOWLEDGE_SERVICE_URL` is invalid, or database migration failed | `docker compose logs qa migrate-qa` |
| File uploads disappear after restart | `FILE_STORAGE_BACKEND=memory` was used | Set `FILE_STORAGE_BACKEND=local` |
| MinIO buckets missing | `minio-init` did not complete | `docker compose logs minio-init` |

## Stop And Reset

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml down
docker compose --env-file deploy/.env -f deploy/docker-compose.yml down -v
```

The `-v` form removes local Postgres, Qdrant, MinIO, file, and QA workspace volumes.
