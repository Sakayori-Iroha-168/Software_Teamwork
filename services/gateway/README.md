# Gateway Service

`services/gateway` is the public backend entrypoint for frontend `/api/v1/**`
requests. This implementation slice covers the A-02 authentication context
baseline:

- auth-owned user/session public routes,
- opaque bearer token hashing,
- Redis session cache lookup/write/delete,
- current-user response from the gateway session cache,
- downstream identity header construction.

It intentionally does not implement A-03 active route proxy coverage yet.

## Local Commands

Run from this directory:

```bash
go test ./...
go build ./cmd/server
```

Run the service:

```bash
GATEWAY_TOKEN_HASH_SECRET=dev-only-change-me go run ./cmd/server
```

## Environment

| Variable | Default | Description |
| --- | --- | --- |
| `GATEWAY_HTTP_ADDR` | `:8080` | HTTP listen address. |
| `GATEWAY_AUTH_BASE_URL` | `http://localhost:8081` | Auth service internal base URL. |
| `GATEWAY_AUTH_SERVICE_TOKEN` | empty | Optional service-to-service token sent to auth. |
| `GATEWAY_REDIS_ADDR` | `localhost:6379` | Redis address for gateway session cache. |
| `GATEWAY_REDIS_PASSWORD` | empty | Redis password. |
| `GATEWAY_REDIS_DB` | `0` | Redis database index. |
| `GATEWAY_TOKEN_HASH_SECRET` | required | HMAC secret for opaque access token hashes. |
| `GATEWAY_TOKEN_HASH_KEY_VERSION` | `v1` | Hash key version used in `hmac-sha256:<version>:<hex>`. |
| `GATEWAY_AUTH_TIMEOUT` | `5s` | Auth service HTTP timeout. |
| `GATEWAY_MAX_REQUEST_BYTES` | `1048576` | Maximum JSON request body size. |
| `GATEWAY_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout. |

Redis keys use:

```text
gateway:session:<accessTokenHash>
```

The cache value never stores the raw access token.
