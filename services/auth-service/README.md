# Auth & User Service (Go)

Source of truth for users. Issues RS256-signed JWT access tokens + opaque
refresh tokens (tracked in Redis). Other services verify access tokens with the
public key alone (`keys/jwt_public.pem`).

## Layout
```
cmd/server/main.go        wiring + graceful shutdown
internal/config           env config
internal/db               pgx pool, redis client, embedded migrations
internal/token            RS256 sign/verify + opaque refresh tokens
internal/model            User
internal/repo             user SQL (pgx)
internal/service          register / login / refresh / logout / profile
internal/middleware        Bearer-token auth middleware
internal/handler          HTTP handlers (JSON)
internal/router           chi routes
```

## Run locally (needs Postgres + Redis running — see root README)
```bash
go mod tidy            # one-time: resolve deps + write go.sum
cp .env.example .env   # adjust if needed
go run ./cmd/server
```

## Endpoints
| Method | Path             | Auth   | Body / notes |
|--------|------------------|--------|--------------|
| GET    | `/health`        | —      | liveness |
| POST   | `/auth/register` | —      | `{email, phone?, password, display_name}` → 201 user |
| POST   | `/auth/login`    | —      | `{email, password}` → `{user, tokens}` |
| POST   | `/auth/refresh`  | —      | `{refresh_token}` → new token pair (rotates) |
| POST   | `/auth/logout`   | —      | `{refresh_token}` → 204 |
| GET    | `/users/me`      | Bearer | current profile |
| PATCH  | `/users/me`      | Bearer | `{display_name, avatar_url?}` |
| GET    | `/users/{id}`    | internal | sibling-service lookup |
| GET    | `/users`         | admin  | `?page&limit` → `{items,page,limit,total}` |
| PATCH  | `/users/{id}/status` | admin | `{status:"active"\|"banned"}` (ban/unban) |

## Quick smoke test
```bash
curl -s localhost:8001/auth/register -d '{"email":"a@b.com","password":"secret1","display_name":"An"}'
curl -s localhost:8001/auth/login    -d '{"email":"a@b.com","password":"secret1"}'
# copy access_token from the response:
curl -s localhost:8001/users/me -H "Authorization: Bearer <ACCESS_TOKEN>"
```
