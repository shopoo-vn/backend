-- Schema "chat" itself is created by infra/postgres/init-db.sql (the chat_svc
-- role has CREATE-in-schema, not CREATE-schema). This migration creates tables.

CREATE TABLE IF NOT EXISTS chat.conversations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  listing_id      UUID,
  buyer_id        UUID NOT NULL,
  seller_id       UUID NOT NULL,
  last_message_at TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (buyer_id, seller_id, listing_id)
);

CREATE TABLE IF NOT EXISTS chat.messages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES chat.conversations(id) ON DELETE CASCADE,
  sender_id       UUID NOT NULL,
  body            TEXT NOT NULL,
  client_msg_id   TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  read_at         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_messages_conv_created
  ON chat.messages (conversation_id, created_at);

-- Dedupe key for idempotent retries (clientMsgId is unique per conversation).
CREATE UNIQUE INDEX IF NOT EXISTS uq_messages_conv_client_msg
  ON chat.messages (conversation_id, client_msg_id)
  WHERE client_msg_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_conversations_buyer ON chat.conversations (buyer_id);
CREATE INDEX IF NOT EXISTS idx_conversations_seller ON chat.conversations (seller_id);
