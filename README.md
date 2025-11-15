# Go API Gateway – Phase 1

Phase 1 delivers a lightweight Go reverse proxy that fronts two mock services. The gateway handles path-based routing (`/a/*`, `/b/*`, `/healthz`, `/`), injects `X-Request-ID`, logs every request, and ships with Docker assets plus a Makefile for local workflows.

## Features
- Reverse-proxy routes for Service A (`/a/*`) and Service B (`/b/*`); prefixes are stripped before forwarding.
- `/healthz` for liveness checks and `/` returning gateway metadata.
- Request ID middleware propagates `X-Request-ID` downstream; logging middleware captures method/path/duration/rid.
- Dockerfile + docker-compose stack build the gateway and both mock services with Go 1.22.
- Makefile targets (`build`, `run`, `test`, `docker-build`, `up`, `down`, `logs`, etc.).

## Project Layout
- `cmd/gateway`: main entry point.
- `internal/config|health|middleware|proxy`: configuration loader, health handler, middleware stack, and reverse-proxy helpers.
- `mock/service-a`, `mock/service-b`: Go HTTP servers returning JSON reflecting request details.
- `pkg/version`: placeholder for build metadata.
- Top-level `Dockerfile`, `docker-compose.yml`, `.env.example`, `Makefile`.
- `docs/ARCHITECTURE.md`: detailed explanation of request flow and architecture.

## Quickstart

```bash
make up
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/a/hello
curl -s http://localhost:8080/b/hello
make logs   # follow gateway logs
```

Tear everything down with:

```bash
make down
# or
docker compose down
```

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `GATEWAY_PORT` / `PORT` | `8080` | Port exposed by the gateway. |
| `BACKEND_A_URL` / `BACKEND_A` | `http://service-a:8080` | Target URL for `/a/*`. |
| `BACKEND_B_URL` / `BACKEND_B` | `http://service-b:8080` | Target URL for `/b/*`. |
| `PORT` (mock services) | `8080` | Override mock service listener port. |

Set them via `.env`, shell exports, or compose environment section. The gateway’s `config.Load()` picks the first non-empty value from each list.

## Docker Compose

- `gateway`: builds from `.` and exposes `8080:8080`.
- `service-a` / `service-b`: build from `./mock/service-*`, listen on port 8080 inside the Docker network.
- Defaults wire the gateway to `service-a:8080` and `service-b:8080`.

Run `docker compose up --build -d` to compile all binaries and start the stack; use `docker logs -f go-api-gateway-gateway-1` (and corresponding mock containers) to inspect traffic.

## Local Development

```bash
go run ./cmd/gateway
# With custom backends:
BACKEND_A_URL=http://localhost:9001 BACKEND_B_URL=http://localhost:9002 go run ./cmd/gateway
```

Start the mock services individually with `go run ./mock/service-a` or `./mock/service-b`, or point the gateway at any other HTTP services.

## Next Steps

Phase 2 ideas include dynamic route tables, resilience (timeouts, retries, circuit breakers), metrics (Prometheus), structured logging, and authentication middleware. See `docs/ARCHITECTURE.md` for the current design baseline.
