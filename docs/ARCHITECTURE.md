# Go API Gateway – Architecture & Flow

This document walks through the major pieces of the Phase 1 gateway, how requests are routed to the mock services, and what happens when you run the Docker stack.

## High-Level Overview

- Single Go binary exposes `/healthz`, `/a/*`, `/b/*`, and `/`.
- Requests pass through two middlewares: `RequestID` (injects `X-Request-ID`) and `Logger`.
- `/a/*` traffic proxies to mock Service A, `/b/*` to mock Service B, using `net/http/httputil`.
- Mock services are lightweight Go HTTP servers that echo request details.
- Docker Compose builds all three binaries with Go 1.22 and wires their networking so the gateway talks to backends via service names.

```
client -> gateway (middleware, routing) -> reverse proxy -> mock service
```

## Gateway Components

### Entry Point – `cmd/gateway/main.go`

1. Load configuration via `config.Load()`.
2. Register handlers:
   - `/healthz` → `internal/health.Handler`
   - `/a/` → `internal/proxy.PathPrefixProxy("/a", cfg.BackendA)`
   - `/b/` → same for backend B
   - `/` → static JSON (`{"name":"api-gateway","status":"ok"}`)
3. Wrap the mux with `middleware.RequestID` and `middleware.Logger`.
4. Start `http.Server` on `fmt.Sprintf(":%d", cfg.Port)`.

### Configuration – `internal/config/config.go`

- Reads env vars with fallbacks that match Docker defaults:

| Setting | Env keys (first match wins) | Default |
| --- | --- | --- |
| Port | `GATEWAY_PORT`, `PORT` | `8080` |
| Backend A | `BACKEND_A_URL`, `BACKEND_A` | `http://service-a:8080` |
| Backend B | `BACKEND_B_URL`, `BACKEND_B` | `http://service-b:8080` |

- `parsePort` logs and falls back to 8080 if the env value is invalid.

### Middleware – `internal/middleware/logging.go`

- `RequestID` generates a UUID, sets it on the request and response headers as `X-Request-ID`.
- `Logger` records `method`, `path`, `requestId`, and the duration for every response.

### Health Endpoint – `internal/health/health.go`

Returns `{"status":"healthy"}` with `Content-Type: application/json`. Used by `/healthz` and the quick sanity checks in the README.

### Proxy Layer – `internal/proxy/proxy.go`

`PathPrefixProxy(prefix, target)`:

1. Parses `target` into a `url.URL`.
2. Builds a `httputil.NewSingleHostReverseProxy`.
3. For each request:
   - Trim the route prefix once (`strings.TrimPrefix`), so `/a/hello` → `/hello` before forwarding.
   - Call the reverse proxy, which rewrites the scheme/host to the backend and streams the response back to the client.

Because the proxy handler runs behind the logging middleware, every proxied request receives an `X-Request-ID` header and gets logged at the gateway.

## Mock Services

Both mocks are plain Go binaries with one handler at `/`.

| Service | Port env | Default port | Response structure |
| --- | --- | --- | --- |
| Service A (`mock/service-a/main.go`) | `PORT` | 8080 | `service`, `message`, `request_id`, `request_uri`, and a map of headers. Logs method/path/rid per request. |
| Service B (`mock/service-b/main.go`) | `PORT` | 8080 | `service`, `message`, `request_id`, `request_uri`. Logs method/path/rid per request. |

Their Dockerfiles (`mock/service-*/Dockerfile`) use `golang:1.22-alpine`, compile the binaries, expose port 8080, and run the resulting executable.

## Request Lifecycle

1. **Client request** hits the gateway (e.g., `GET /a/hello`).
2. **Request ID middleware** attaches `X-Request-ID`.
3. **Logging middleware** wraps the handler to time the request.
4. **Routing**:
   - `/healthz` returns JSON immediately.
   - `/a/*` or `/b/*` invoke the proxy; the prefix is stripped and the request is sent to `service-a` or `service-b`.
   - Any other path returns the root JSON.
5. **Reverse proxy** forwards the request over the Docker network (`service-a:8080`, `service-b:8080`) and streams the response.
6. **Response** flows back through the middleware (which writes the `X-Request-ID` header) to the client.
7. **Logging** prints a single line on the gateway with method/path/requestId/duration. Mock services log their own per-request line.

## Running with Docker Compose

`docker-compose.yml` defines three services:

| Service | Build context | Ports | Environment |
| --- | --- | --- | --- |
| `gateway` | `.` | `8080:8080` | `GATEWAY_PORT=8080`, `BACKEND_A_URL=http://service-a:8080`, `BACKEND_B_URL=http://service-b:8080` |
| `service-a` | `./mock/service-a` | internal only (exposed via gateway) | `PORT=8080` (default inside image) |
| `service-b` | `./mock/service-b` | same | same |

Workflow:

1. `docker compose up --build -d`
2. Hit the endpoints:
   ```bash
   curl -s http://localhost:8080/healthz
   curl -s http://localhost:8080/a/hello
   curl -s http://localhost:8080/b/hello
   ```
3. Inspect logs:
   ```bash
   docker logs -f go-api-gateway-gateway-1
   docker logs -f go-api-gateway-service-a-1
   docker logs -f go-api-gateway-service-b-1
   ```
4. Tear down with `docker compose down`.

Because the gateway only depends on the Go standard library (plus `github.com/google/uuid` for request IDs), the same flow also works outside Docker: run `go run ./cmd/gateway` and point `BACKEND_A_URL/B` to whatever services you want to proxy.
