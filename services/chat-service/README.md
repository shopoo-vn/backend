# chat-service

Real-time 1-1 buyer↔seller chat for the shopoo marketplace. Express handles REST
history/conversations; Socket.io handles live messaging, presence, typing and read
receipts. TypeScript throughout.

- **Port:** 8006
- **DB schema:** `chat` (role `chat_svc`)
- **Presence:** Redis (`online:<userId>` → set of socketIds)
- **Scale:** `@socket.io/redis-adapter` (emits reach sockets on other instances)
- **Events out:** `chat.message.created` → `marketplace.events`

## Run locally

```bash
cp .env.example .env      # adjust if needed
npm install
npm run build
npm start                 # or: npm run start:dev
```

Migrations run automatically at startup (ordered `.sql` files in `src/db/migrations`,
tracked in `chat.schema_migrations`).

## Auth

All REST endpoints except `/health` require `Authorization: Bearer <accessToken>`.
WebSocket clients pass the same token in `handshake.auth.token`. Tokens are verified
with the Auth Service **public key only** (RS256, issuer `marketplace-auth`, expiry
enforced). This service never holds a private key.

## REST endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | none | Liveness → `{"status":"ok"}` |
| GET | `/conversations` | Bearer | Caller's conversations with `lastMessage` + `unread` count |
| POST | `/conversations` | Bearer | Get-or-create. Body `{ sellerId, listingId? }`; caller is the buyer → `201` conversation |
| GET | `/conversations/:id/messages?before=&limit=` | Bearer (participant) | Paginated history. `before` = ISO-8601 keyset cursor, `limit` ≤ 100 (default 20) → `{ items, limit }` |

Errors use the project shape: `{"error":"message"}`.

## WebSocket events

Connect with `io(url, { auth: { token } })`. Each conversation is room `conv:<id>`;
the server authorizes the caller is a participant before any join/emit. Messages are
persisted to Postgres **before** they are emitted, and deduped by `clientMsgId`.

### client → server

| Event | Payload | Ack | Description |
|-------|---------|-----|-------------|
| `message:send` | `{ conversationId, body, clientMsgId }` | `{ ok, message?, error? }` | Persist + fan out to the room. `senderId` is taken from the verified socket, never the payload. |
| `typing` | `{ conversationId }` | — | Relayed to the other participant. |
| `message:read` | `{ conversationId }` | — | Marks the peer's messages read; notifies the room. |

### server → client

| Event | Payload | Description |
|-------|---------|-------------|
| `message:new` | `{ message }` | A new message in a joined conversation. |
| `message:delivered` | `{ conversationId, messageId, clientMsgId }` | Server persisted the sender's message. |
| `presence:update` | `{ userId, online }` | A conversation peer came online / went offline. |
| `typing` | `{ conversationId, userId }` | A peer is typing. |
| `message:read` | `{ conversationId, userId }` | A peer read the conversation. |
| `error` | `{ error }` | Project-shape error. |

## Published events

| Routing key | When | Data |
|-------------|------|------|
| `chat.message.created` | After a new message is committed | `{ conversationId, messageId, senderId, recipientId }` |

Envelope: `{ eventId, type, occurredAt, data }` on topic exchange `marketplace.events`.

## Config (env, prefix `CHAT_`)

| Var | Default | Notes |
|-----|---------|-------|
| `CHAT_HTTP_PORT` | `8006` | |
| `CHAT_DB_URL` | — | `postgres://chat_svc:chat_pw@host:5432/marketplace` |
| `CHAT_REDIS_URL` | — | `redis://host:6379` |
| `CHAT_RABBITMQ_URL` | — | `amqp://guest:guest@host:5672` |
| `RABBITMQ_EXCHANGE` | `marketplace.events` | |
| `CHAT_JWT_PUBLIC_KEY_PATH` | — | path to `keys/jwt_public.pem` (e.g. `../../keys/jwt_public.pem`) |
| `CHAT_JWT_ISSUER` | `marketplace-auth` | |
| `CHAT_CORS_ORIGINS` | `*` | Comma-separated allow-list |
