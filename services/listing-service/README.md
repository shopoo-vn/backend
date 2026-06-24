# Listing & Search Service (NestJS)

The heart of the marketplace: listing CRUD, category browsing, advanced filtering
and PostgreSQL full-text search. Built per `nestjs-service-patterns` +
`marketplace-conventions`.

## Layout
```
src/
  main.ts                 bootstrap (ValidationPipe, exception filter, CORS, shutdown)
  app.module.ts           ConfigModule (+Joi) + TypeORM + feature modules
  health.controller.ts    GET /health
  config/                 typed config + env validation
  common/                 exception filter, pagination dto, numeric transformer, auth
  database/               TypeORM data-source + migrations (tsvector + GIN + seed)
  messaging/              RabbitMQ publish/consume (marketplace.events)
  category/               categories (browse + admin create)
  listing/                listings (CRUD, search, moderation consumer)
```

## Auth
Verifies **RS256** access tokens with the Auth Service **public key**
(`keys/jwt_public.pem`). Browse/search are public; create/update/delete require a
Bearer token (owner or admin); category create requires `role=admin`.

## Endpoints
| Method | Path             | Auth   | Notes |
|--------|------------------|--------|-------|
| GET    | `/health`        | —      | liveness |
| GET    | `/categories`    | —      | list categories |
| POST   | `/categories`    | admin  | `{name, slug, parentId?}` |
| GET    | `/listings`      | —      | filters: `q, categoryId, minPrice, maxPrice, location, condition, sort, page, limit` → `{items,page,limit,total}` (active only) |
| GET    | `/listings/:id`  | —      | one listing |
| POST   | `/listings`      | Bearer | `{title, description?, price, categoryId, condition, location?, mediaIds?}` → status `pending`, emits `listing.created` |
| PATCH  | `/listings/:id`  | owner/admin | partial update, emits `listing.updated` |
| DELETE | `/listings/:id`  | owner/admin | 204, emits `listing.deleted` |
| GET    | `/admin/listings` | admin | all statuses; `?status&q&categoryId&minPrice&maxPrice&location&condition&sort&page&limit` → `{items,page,limit,total}` |

**Events:** publishes `listing.created|updated|deleted`; consumes `listing.approved|rejected`
(from Admin Service) to flip status `pending → active|rejected` (idempotent).

## Run locally
Requires Postgres (with the `listing` schema from `infra/postgres/init-db.sql`) and
RabbitMQ running — easiest via `docker compose up -d postgres rabbitmq` in `backend/`.
```bash
npm install
cp .env.example .env
npm run start:dev        # migrations run automatically on boot
```

## Smoke test
```bash
curl -s localhost:8004/health
curl -s localhost:8004/categories
TOKEN=...   # access token from auth-service login
curl -s localhost:8004/listings -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"iPhone 13 128GB","price":9500000,"categoryId":"<cat-uuid>","condition":"like_new"}' \
  -H 'Content-Type: application/json'
curl -s "localhost:8004/listings?q=iphone&sort=price_asc"
```

> Note: `GET /listings` returns only `active` listings. A freshly created listing is
> `pending` until Admin approves it (M4). For dev you can approve by emitting
> `listing.approved` or updating the row directly.
