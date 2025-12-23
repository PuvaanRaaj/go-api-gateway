# Go API Gateway – Phase 2

Phase 2 evolves the lightweight reverse proxy into an authenticated gateway: `/a/*` and `/b/*` remain path-based routes, but traffic now requires either a JWT (issued by `/auth/login`) or a valid API key stored in PostgreSQL/Supabase. Request logging, request IDs, Docker assets, and mock services remain from Phase 1.

## Features
- Reverse proxy for Service A (`/a/*`) and Service B (`/b/*`) with stripped prefixes.
- `/auth/login` endpoint plus middleware enforcing JWT Bearer tokens.
- API key authentication (`X-API-Key`) backed by PostgreSQL with seed data.
- Postgres/Supabase integration with Go-based migrations.
- Request/response logging + `X-Request-ID` propagation.
- Docker Compose stack (`gateway`, `service-a`, `service-b`, `db`) and handy Makefile targets.

## Project Layout
- `cmd/gateway`: main entry point.
- `cmd/migrate`: helper to apply SQL migrations.
- `internal/config|health|middleware|proxy`: configuration loader, health handler, middleware stack, and reverse-proxy helpers.
- `internal/auth`: login handler, JWT helpers, identity context helpers.
- `internal/database`, `internal/store`: Postgres connection + user/API-key data access.
- `mock/service-a`, `mock/service-b`: Go HTTP servers returning JSON reflecting request details.
- `pkg/version`: placeholder for build metadata.
- `migrations/`: SQL files for creating `users` + `api_keys` and seeding demo records.
- Top-level `Dockerfile`, `docker-compose.yml`, `.env.example`, `Makefile`.
- `docs/ARCHITECTURE.md` and `docs/PHASE2_AUTH.md`: architecture deep dives.

## Quickstart

```bash
cp .env.example .env        # optional overrides
make up                     # builds + starts gateway, mocks, and Postgres
make migrate                # apply migrations + seed demo data

# Obtain JWT
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@puvaan.dev","password":"password"}' | jq -r .token)

# Call proxied services
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/a/hello | jq
curl -s -H "X-API-Key: demo-key-123" http://localhost:8080/b/hello | jq

make logs                   # tail gateway logs
```

Tear everything down with:

```bash
make down
```

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `GATEWAY_PORT` / `PORT` | `8080` | Port exposed by the gateway. |
| `BACKEND_A_URL` / `BACKEND_A` | `http://service-a:8080` | Target URL for `/a/*`. |
| `BACKEND_B_URL` / `BACKEND_B` | `http://service-b:8080` | Target URL for `/b/*`. |
| `DATABASE_URL` / `SUPABASE_DB_URL` | `postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable` | Postgres DSN used by the gateway and migrations. Works with Supabase connection strings as well (Compose overrides to `db`). |
| `JWT_SECRET` | `dev-secret` | Symmetric key for signing HS256 JWTs. |
| `API_KEY_HEADER` | `X-API-Key` | Header checked for API key authentication. |
| `TOKEN_TTL` | `1h` | Lifetime for issued JWTs. |
| `PORT` (mock services) | `8080` | Override mock service listener port for Service A/B. |

Set them via `.env`, shell exports, or the Compose environment section. `config.Load()` picks the first non-empty value per setting.

## Docker Compose

- `gateway`: Go reverse proxy + auth server (port `8080`).
- `service-a` / `service-b`: mock JSON services (internal port `8080`).
- `db`: Postgres 15 with persistent volume `pgdata`.

Switching to Supabase? Set `DATABASE_URL` (or `SUPABASE_DB_URL`) to the supplied connection string and disable/comment out the `db` service.

Run `docker compose up --build -d` to compile all binaries and start the stack; use `docker logs -f go-api-gateway-gateway-1` (and corresponding mock containers) to inspect traffic.

## Local Development

```bash
DATABASE_URL=postgres://... JWT_SECRET=supersecret go run ./cmd/gateway

# Migrations
DATABASE_URL=postgres://... go run ./cmd/migrate
```

Start the mock services individually with `go run ./mock/service-a` or `./mock/service-b`, or point the gateway at any other HTTP services by overriding the `BACKEND_*` URLs.

## Authentication

1. **JWT** – Clients call `POST /auth/login` with JSON `{ "email": "...", "password": "..." }`. Valid credentials return `{ "token": "...", "expires_at": "..." }`. Include `Authorization: Bearer <token>` on subsequent calls.
2. **API Key** – Supply `X-API-Key: <key>` (header configurable via `API_KEY_HEADER`). Keys live in the `api_keys` table with `revoked` toggles.
3. **Protected routes** – Everything except `/healthz` and `/auth/login` requires authentication. Both mechanisms attach the identity to the request context so downstream handlers can use it later.

Demo records (created by `make migrate`):
- User: `demo@puvaan.dev` / `password`
- API key: `demo-key-123`

## Database & Migrations

- `cmd/migrate` reads every SQL file under `migrations/` and executes them in lexical order.
- `make migrate` runs the helper against the current `DATABASE_URL`.
- Schema includes `users` (bcrypt hashes) and `api_keys` (per-user keys with revoke flag). The SQL is Postgres-compatible and works unchanged on Supabase.

## Next Steps

Phase 3 wishlist:

1. Dynamic route tables driven by config/migrations.
2. Circuit breaking, retries, and per-route timeouts.
3. Metrics (Prometheus) + structured logging (slog/zap).
4. Token refresh & role-based authorization.
5. Rate limiting and API key self-service UI.

Details for the newly added authentication stack live in `docs/PHASE2_AUTH.md`.
