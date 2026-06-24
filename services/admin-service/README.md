# Admin Service (NestJS)

Admin-only back office for the marketplace: a dashboard, the listing **moderation
queue**, and user **reports**. Built per `nestjs-service-patterns` +
`marketplace-conventions`.

## Layout
```
src/
  main.ts                 bootstrap (ValidationPipe, exception filter, CORS, shutdown)
  app.module.ts           ConfigModule (+Joi) + TypeORM + feature modules
  health.controller.ts    GET /health
  config/                 typed config + env validation
  common/                 exception filter, pagination dto, auth (JWT RS256, guards)
  database/               TypeORM data-source + migration (admin schema)
  messaging/              RabbitMQ publish/consume (marketplace.events)
  moderation/             moderation queue (consumes listing.created, approve/reject)
  reports/                user reports (list + resolve)
  dashboard/              aggregate counts
```

## Auth
Every `/admin/*` route requires an **RS256** Bearer access token whose `role=admin`
(`JwtAuthGuard` + `RolesGuard` + `@Roles('admin')`). Tokens are verified with the Auth
Service **public key** only (`keys/jwt_public.pem`). `GET /health` is open.

## Endpoints
| Method | Path                                  | Auth  | Notes |
|--------|---------------------------------------|-------|-------|
| GET    | `/health`                             | —     | liveness → `{status:"ok"}` |
| GET    | `/admin/dashboard`                    | admin | `{pending, approved, rejected, reportsOpen}` |
| GET    | `/admin/moderation/queue?page&limit`  | admin | paginated pending items → `{items,page,limit,total}` |
| POST   | `/admin/moderation/:listingId/approve`| admin | sets `approved`, `reviewedBy`; emits `listing.approved` |
| POST   | `/admin/moderation/:listingId/reject` | admin | body `{reason}`; sets `rejected`; emits `listing.rejected` |
| GET    | `/admin/reports?page&limit`           | admin | paginated reports → `{items,page,limit,total}` |
| POST   | `/admin/reports/:id/resolve`          | admin | sets report `resolved` |

## Events
| Direction | Routing key        | Payload | Effect |
|-----------|--------------------|---------|--------|
| consume   | `listing.created`  | `{listingId, sellerId, title}` | upsert `moderation_items` as `pending` (idempotent) |
| publish   | `listing.approved` | `{listingId}` | Listing Service flips status → `active` |
| publish   | `listing.rejected` | `{listingId, reason}` | Listing Service flips status → `rejected` |

Consumes via durable queue `admin-service.moderation`. Events publish **after** the DB
commit; consumer acks on success, nacks (no-requeue) on failure.

## Data (schema `admin`)
- `moderation_items(listing_id PK, seller_id, title, status pending|approved|rejected,
  reviewed_by, reason, created_at, updated_at)`
- `reports(id PK, target_type, target_id, reporter_id, reason, status, created_at)`

The `admin` schema + `admin_svc` role are created by `infra/postgres/init-db.sql`; this
service's migration creates the tables (runs automatically on boot).

## Run locally
Requires Postgres (with the `admin` schema) and RabbitMQ — easiest via
`docker compose up -d postgres rabbitmq` in `backend/`.
```bash
npm install
cp .env.example .env
npm run start:dev        # migrations run automatically on boot
```

## Smoke test
```bash
curl -s localhost:8005/health
TOKEN=...   # admin access token from auth-service login
curl -s localhost:8005/admin/dashboard -H "Authorization: Bearer $TOKEN"
curl -s "localhost:8005/admin/moderation/queue?page=1&limit=20" -H "Authorization: Bearer $TOKEN"
curl -s -X POST localhost:8005/admin/moderation/<listing-uuid>/approve -H "Authorization: Bearer $TOKEN"
curl -s -X POST localhost:8005/admin/moderation/<listing-uuid>/reject -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"reason":"prohibited item"}'
```
