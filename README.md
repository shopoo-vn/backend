# Backend — Marketplace Đồ Điện Tử

Polyglot microservices (repo `backend/` trong workspace `shopoo/`). Kế hoạch:
- [`../implementation_plan_marketplace.md`](../implementation_plan_marketplace.md) — overview & service split
- [`../implementation_plan_backend.md`](../implementation_plan_backend.md) — services, events, DB, milestones

## Status
| Milestone | What | State |
|---|---|---|
| M0 | Infra (Postgres / Redis / RabbitMQ / MinIO) + skeleton | ✅ done |
| M1 | Auth & User Service (Go, RS256 JWT) | ✅ done · build verified |
| M2 | Listing & Search (NestJS) — CRUD + full-text + events | ✅ done · build verified |
| M3 | Media Service (Go) — upload/resize → MinIO | ✅ code · ⚠️ needs `go mod tidy && go build` |
| M4 | Admin Service (NestJS) — moderation + dashboard | ✅ done · build verified |
| M5 | Chat Service (Express + Socket.io) — realtime | ✅ done · build verified |
| M6 | Notification Service (Go) — FCM worker | ✅ code · ⚠️ needs `go mod tidy && go build` |

> Go services (auth, media, noti) compile only after `go mod tidy` on a machine with Go 1.22 — Go is not installed in the dev box where they were authored. NestJS/Express services were `npm run build`-verified.

## Layout
```
services/
  auth-service/        Go — auth, users, JWT (RS256)      [done, build verified]
  listing-service/     NestJS — listings + search          [done, build verified]
  admin-service/       NestJS — dashboard + moderation     [done, build verified]
  chat-service/        Express + Socket.io — realtime      [done, build verified]
  media-service/       Go — image upload/resize → MinIO    [code, needs go build]
  noti-service/        Go — FCM worker (RabbitMQ consumer) [code, needs go build]
infra/postgres/init-db.sql   schemas + per-service roles (run by Postgres on first init)
keys/                  RS256 keypair (gitignored)
scripts/gen-keys.*     regenerate the keypair
docker-compose.yml     all 6 services + Postgres, Redis, RabbitMQ, MinIO
```

## Prerequisites
- **Docker Desktop** (Compose v2). Each service builds inside its own container, so you don't need Go/Node locally to run the stack — only **OpenSSL** (or the keypair already in `keys/`).

## Quickstart
```powershell
# 1. Generate the RS256 keypair (skip if keys/ already present)
pwsh scripts/gen-keys.ps1          # or: bash scripts/gen-keys.sh

# 2. (optional) override infra creds
Copy-Item .env.example .env

# 3. Build + start everything (infra + all 6 services)
docker compose up -d --build       # first run is slow: pulls images + builds 6 services

# 4. Status / logs
docker compose ps
docker compose logs -f listing-service
```

## Smoke test
```powershell
curl.exe http://localhost:8001/health
curl.exe -X POST http://localhost:8001/auth/register -H "Content-Type: application/json" -d "{\"email\":\"a@b.com\",\"password\":\"secret1\",\"display_name\":\"An\"}"
curl.exe http://localhost:8004/categories
```
Ports: auth 8001 · media 8002 · noti 8003 · listing 8004 · admin 8005 · chat 8006. Infra: Postgres 5432 · Redis 6379 · RabbitMQ 5672 (UI 15672) · MinIO 9000 (console 9001).

## Developing a single service (hot reload)
Run only the infra in Docker, then run the service you're editing locally against it:
```powershell
docker compose up -d postgres redis rabbitmq minio
cd services/listing-service ; npm install ; npm run start:dev   # :8004
```
The `.env.example` files default to `localhost`, matching the published infra ports.

> ⚠️ Dev-only credentials live in `init-db.sql`, `docker-compose.yml`, and the
> `.env.example` files. Override before any real deployment; keep `keys/` and `.env` out of git.
