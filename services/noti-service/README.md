# Notification Service (Go)

Background worker that consumes marketplace events from RabbitMQ and sends FCM
push notifications, plus a tiny REST API for managing per-user device tokens.
Verifies RS256 access tokens with the public key alone (`keys/jwt_public.pem`) —
it never holds a private key.

Every push is persisted as a `notifications` row. If no Firebase credentials are
configured (`NOTI_FCM_CREDENTIALS_FILE` unset), push runs in a **logging/no-op**
mode so the service runs end-to-end without Firebase.

## Layout
```
cmd/server/main.go        wiring + graceful shutdown
internal/config           env config (NOTI_*)
internal/db               pgx pool, redis client, embedded migrations
internal/token            RS256 verify-only (public key)
internal/model            DeviceToken, Notification
internal/repo             device-token + notification SQL (pgx)
internal/push             FCM sender (firebase.google.com/go/v4) + no-op fallback
internal/service          device-token mgmt + event → notification + idempotency
internal/event            RabbitMQ consumer (topic exchange, durable queue)
internal/middleware       Bearer-token auth middleware
internal/handler          HTTP handlers (JSON)
internal/router           chi routes
```

## Run locally (needs Postgres + Redis + RabbitMQ running — see root README)
```bash
go mod tidy            # one-time: resolve deps + write go.sum
cp .env.example .env   # adjust if needed
go run ./cmd/server
```

## REST endpoints
| Method | Path               | Auth   | Body / notes |
|--------|--------------------|--------|--------------|
| GET    | `/health`          | —      | `{"status":"ok"}` liveness |
| POST   | `/devices`         | Bearer | `{token, platform}` → 201 device token (upsert) |
| DELETE | `/devices/{token}` | Bearer | remove the caller's token → 204 (404 if not found) |

## Consumed events (exchange `marketplace.events`, queue `noti-service.events`)
| Routing key             | Action |
|-------------------------|--------|
| `chat.message.created`  | push "Bạn có tin nhắn mới" to `data.recipientId` |
| `listing.approved`      | notify the seller (`data.sellerId`/`userId` if present) that the listing was approved |
| `listing.rejected`      | notify the seller that the listing was rejected (includes `reason`) |

Idempotency: each event's `eventId` is claimed via Redis `SETNX processed:<eventId>`
(TTL `NOTI_PROCESSED_TTL_MIN`). A first-seen event is processed and **acked**; a
duplicate is skipped and acked. On handler failure the message is **nacked
without requeue** (→ DLQ) and the Redis marker is released so the event can be
replayed.

## Notes
- This service publishes nothing; it is a pure consumer + small REST API.
- The current admin-service `listing.approved/rejected` payloads carry only
  `listingId`. Since noti-service cannot read the listing schema, it notifies the
  seller when the payload includes `sellerId`/`userId`, otherwise it logs and
  acks (the notification row + push are skipped). Forward-compatible: add
  `sellerId` to the publisher and pushes start flowing with no change here.
