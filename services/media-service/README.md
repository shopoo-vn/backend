# Media Service (Go)

Uploads product images, resizes/compresses them into multiple sizes using a
**bounded goroutine worker pool**, stores the objects in **MinIO** (S3-compatible)
and persists metadata in the Postgres `media` schema. Built per
`golang-service-patterns` + `marketplace-conventions`, mirroring `auth-service`.

## Layout
```
cmd/server/main.go        wiring + graceful shutdown
internal/config           typed env config (MEDIA_* vars)
internal/db               pgx pool + embedded migrations
internal/token            RS256 verify-ONLY (Auth Service public key)
internal/model            Media
internal/repo             media SQL (pgx)
internal/storage          MinIO client (bucket ensure / put / remove / url)
internal/imageproc        decode once + bounded-pool resize/encode
internal/service          upload / get / delete orchestration
internal/middleware       Bearer-token auth middleware
internal/handler          HTTP handlers (JSON + multipart)
internal/router           chi routes
```

## Auth
Verifies **RS256** access tokens with the Auth Service **public key only**
(`keys/jwt_public.pem`, env `MEDIA_JWT_PUBLIC_KEY_PATH`). Issuer and `alg=RS256`
are enforced. This service never loads a private key. `GET /media/:id` is public
(object URLs are public); upload and delete require a Bearer token.

## Image processing
On upload the original is decoded once, then three variants are produced
concurrently through a counting-semaphore worker pool (size `MEDIA_RESIZE_WORKERS`,
default 4) so concurrent uploads cannot spawn unbounded goroutines:

| Size   | Width        | Notes |
|--------|--------------|-------|
| thumb  | 256px        | downscaled, aspect preserved |
| medium | 1024px       | downscaled, aspect preserved |
| full   | original     | re-encoded / compressed (no upscaling) |

All variants are normalised to JPEG (q85) and stored at
`media/<id>/<size>.jpg` in the `media` bucket.

## Endpoints
| Method | Path             | Auth   | Body / notes |
|--------|------------------|--------|--------------|
| GET    | `/health`        | —      | liveness → `{"status":"ok"}` |
| POST   | `/media/upload`  | Bearer | multipart field `file` (image/*, ≤10MB) → `201 {id, urls:{thumb,medium,full}}` |
| GET    | `/media/:id`     | —      | `{id, owner_id, urls, created_at}` (404 if missing) |
| DELETE | `/media/:id`     | Bearer | owner only → `204`; removes objects + row |

**Events:** none (this service neither publishes nor consumes RabbitMQ events).

## Run locally
Requires Postgres (with the `media` schema + `media_svc` role from
`infra/postgres/init-db.sql`) and MinIO running — easiest via
`docker compose up -d postgres minio` in `backend/`.
```bash
go mod tidy            # one-time: resolve deps + write go.sum
cp .env.example .env   # adjust if needed
go run ./cmd/server
```

## Smoke test
```bash
curl -s localhost:8002/health
TOKEN=...   # access token from auth-service login
ID=$(curl -s localhost:8002/media/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@./some-image.jpg" | jq -r .id)
curl -s localhost:8002/media/$ID
curl -s -X DELETE localhost:8002/media/$ID -H "Authorization: Bearer $TOKEN" -i
```

## Tests
```bash
go test ./...    # imageproc has table-driven tests (variants, no-upscale, cancel, bad input)
```
