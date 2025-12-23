# Phase 2 – Authentication Architecture

This document defines the authentication and authorization features scheduled for Phase 2 of the Go API Gateway.

## Goals

1. Protect gateway routes with JWT bearer tokens.
2. Offer API key authentication for service-to-service access.
3. Persist users and API keys in PostgreSQL (local container or managed Supabase instance).
4. Provide Docker Compose + migrations so local environments can bootstrap easily.

## Components

| Component | Responsibility |
| --- | --- |
| `auth` package (new) | Login handler, JWT generation, shared helpers. |
| `middleware.Auth` | Verifies JWT (primary) or API key (fallback). |
| PostgreSQL | Stores `users`, `api_keys`, and optionally refresh tokens. |
| `/auth/login` endpoint | Issues JWTs after validating credentials (Phase 2 uses static demo users seeded via migrations). |
| Migration files | Create schema + seed data. |
| Docker Compose Postgres service | Local DB for development/testing (configurable to use Supabase connection string instead). |

## High-Level Flow

```
Client -> /auth/login -> validate credentials -> issue JWT (HS256)
Client -> (protected route)
    -> Auth middleware checks Authorization header
        -> JWT valid? allow
        -> else check X-API-Key header against DB
        -> otherwise 401
```

## JWT Authentication

- **Middleware** extracts `Authorization: Bearer <token>`.
- Uses shared secret (env `JWT_SECRET`) or JWKS provider (future upgrade) to validate signature + expiry.
- On success, sets `context.Context` values (`user_id`, `scopes`) for downstream handlers.
- On failure, falls back to API key validation before returning `401 Unauthorized`.

### `/auth/login`

- Accepts JSON payload (e.g., `{ "email": "...", "password": "..." }`).
- Looks up user in Postgres; password storage for Phase 2 can be plaintext demo-only or hashed (bcrypt).
- Issues JWT with claims:
  - `sub`: user ID
  - `email`
  - `exp`: expiration
  - optional `roles`/`scopes`
- Returns `{ "token": "<jwt>", "expires_in": 3600 }`.

## API Key Authentication

- Clients include `X-API-Key: <key>` header.
- Middleware checks active key in Postgres (`api_keys` table) joined with `users`.
- Supports toggling status (`active`, `revoked`) and per-key scopes.
- Useful for internal services where JWT issuance might be cumbersome.

## PostgreSQL / Supabase

- Local setup: add a `postgres` service to `docker-compose.yml` (image `postgres:15-alpine`).
- Provide env vars (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`).
- Gateway connects via `DATABASE_URL`. For Supabase, reuse the provided connection string—schema + migrations are identical because Supabase exposes vanilla Postgres.
- Migrations run via `make migrate` (tooling TBD: `golang-migrate`, `tern`, or simple SQL scripts + helper).

### Schema Outline

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    key TEXT UNIQUE NOT NULL,
    label TEXT,
    scopes TEXT[],
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now()
);
```

Seed demo users + keys for local testing.

## Docker Compose Update

```yaml
services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: gateway
      POSTGRES_PASSWORD: gateway
      POSTGRES_DB: gateway
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  gateway:
    environment:
      - DATABASE_URL=postgres://gateway:gateway@db:5432/gateway?sslmode=disable
      - JWT_SECRET=devsecret
      - API_KEY_HEADER=X-API-Key
    depends_on:
      - db
```

Switching to Supabase:
- Set `DATABASE_URL` to your Supabase connection string.
- Disable the local `db` service (comment or use profile) because Supabase already hosts the Postgres instance.

## Rollout Plan

1. **Data layer** – Add migrations + Postgres service, verify connectivity.
2. **Auth utilities** – Implement JWT creation/validation helpers.
3. **Login endpoint** – `/auth/login` issuing signed tokens.
4. **Middleware** – Enforce auth on proxy routes (`/a/*`, `/b/*`) with opt-out for `/healthz` and `/auth/login`.
5. **API keys** – DB-backed lookup, rotate/revoke support.
6. **Docs & samples** – Update README, add Postman collection or curl examples for login + usage.

This foundation keeps the gateway flexible: run locally with Docker, target Supabase in production, and expand later (refresh tokens, role-based auth, rate limiting, etc.).
