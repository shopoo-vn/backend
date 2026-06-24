import { pool } from '../lib/db';
import type { Conversation, ConversationSummary, Message } from '../types';

interface ConversationRow {
  id: string;
  listing_id: string | null;
  buyer_id: string;
  seller_id: string;
  last_message_at: Date | null;
  created_at: Date;
}

interface MessageRow {
  id: string;
  conversation_id: string;
  sender_id: string;
  body: string;
  client_msg_id: string | null;
  created_at: Date;
  read_at: Date | null;
}

function mapConversation(r: ConversationRow): Conversation {
  return {
    id: r.id,
    listingId: r.listing_id,
    buyerId: r.buyer_id,
    sellerId: r.seller_id,
    lastMessageAt: r.last_message_at ? r.last_message_at.toISOString() : null,
    createdAt: r.created_at.toISOString(),
  };
}

function mapMessage(r: MessageRow): Message {
  return {
    id: r.id,
    conversationId: r.conversation_id,
    senderId: r.sender_id,
    body: r.body,
    clientMsgId: r.client_msg_id,
    createdAt: r.created_at.toISOString(),
    readAt: r.read_at ? r.read_at.toISOString() : null,
  };
}

/** Get-or-create the (buyer, seller, listing) conversation. */
export async function getOrCreateConversation(
  buyerId: string,
  sellerId: string,
  listingId: string | null,
): Promise<Conversation> {
  // ON CONFLICT needs a concrete value; NULL never conflicts on the unique
  // index, so try-insert then fall back to select handles both listing & no-listing.
  const insert = await pool.query<ConversationRow>(
    `INSERT INTO chat.conversations (buyer_id, seller_id, listing_id)
       VALUES ($1, $2, $3)
     ON CONFLICT (buyer_id, seller_id, listing_id) DO NOTHING
     RETURNING *`,
    [buyerId, sellerId, listingId],
  );
  if (insert.rows[0]) {
    return mapConversation(insert.rows[0]);
  }
  const existing = await pool.query<ConversationRow>(
    `SELECT * FROM chat.conversations
       WHERE buyer_id = $1 AND seller_id = $2
         AND listing_id IS NOT DISTINCT FROM $3`,
    [buyerId, sellerId, listingId],
  );
  return mapConversation(existing.rows[0]);
}

export async function findConversationById(id: string): Promise<Conversation | null> {
  const { rows } = await pool.query<ConversationRow>(
    'SELECT * FROM chat.conversations WHERE id = $1',
    [id],
  );
  return rows[0] ? mapConversation(rows[0]) : null;
}

/** A user's conversations with last message + unread count, newest activity first. */
export async function listConversations(userId: string): Promise<ConversationSummary[]> {
  const { rows } = await pool.query<ConversationRow>(
    `SELECT * FROM chat.conversations
       WHERE buyer_id = $1 OR seller_id = $1
       ORDER BY last_message_at DESC NULLS LAST, created_at DESC`,
    [userId],
  );

  const summaries: ConversationSummary[] = [];
  for (const row of rows) {
    const conv = mapConversation(row);
    const last = await pool.query<MessageRow>(
      `SELECT * FROM chat.messages
         WHERE conversation_id = $1
         ORDER BY created_at DESC
         LIMIT 1`,
      [conv.id],
    );
    const unread = await pool.query<{ count: string }>(
      `SELECT COUNT(*)::text AS count FROM chat.messages
         WHERE conversation_id = $1 AND sender_id <> $2 AND read_at IS NULL`,
      [conv.id, userId],
    );
    summaries.push({
      ...conv,
      lastMessage: last.rows[0] ? mapMessage(last.rows[0]) : null,
      unread: parseInt(unread.rows[0]?.count ?? '0', 10),
    });
  }
  return summaries;
}

/** Paginated history (keyset on created_at). before = ISO timestamp cursor. */
export async function listMessages(
  conversationId: string,
  before: string | null,
  limit: number,
): Promise<Message[]> {
  const params: unknown[] = [conversationId];
  let cursor = '';
  if (before) {
    params.push(before);
    cursor = `AND created_at < $${params.length}`;
  }
  params.push(limit);
  const { rows } = await pool.query<MessageRow>(
    `SELECT * FROM chat.messages
       WHERE conversation_id = $1 ${cursor}
       ORDER BY created_at DESC
       LIMIT $${params.length}`,
    params,
  );
  // Return chronological (oldest first) for the client.
  return rows.map(mapMessage).reverse();
}

/**
 * Insert a message and bump conversation.last_message_at in one transaction.
 * Dedupes on (conversation_id, client_msg_id): a retry returns the existing row.
 * Returns { message, created } where created=false means it was a duplicate.
 */
export async function insertMessage(
  conversationId: string,
  senderId: string,
  body: string,
  clientMsgId: string | null,
): Promise<{ message: Message; created: boolean }> {
  const client = await pool.connect();
  try {
    await client.query('SET search_path TO chat');
    await client.query('BEGIN');

    if (clientMsgId) {
      const dup = await client.query<MessageRow>(
        `SELECT * FROM chat.messages
           WHERE conversation_id = $1 AND client_msg_id = $2`,
        [conversationId, clientMsgId],
      );
      if (dup.rows[0]) {
        await client.query('COMMIT');
        return { message: mapMessage(dup.rows[0]), created: false };
      }
    }

    const inserted = await client.query<MessageRow>(
      `INSERT INTO chat.messages (conversation_id, sender_id, body, client_msg_id)
         VALUES ($1, $2, $3, $4)
       RETURNING *`,
      [conversationId, senderId, body, clientMsgId],
    );
    await client.query(
      `UPDATE chat.conversations SET last_message_at = now() WHERE id = $1`,
      [conversationId],
    );
    await client.query('COMMIT');
    return { message: mapMessage(inserted.rows[0]), created: true };
  } catch (err) {
    await client.query('ROLLBACK');
    throw err;
  } finally {
    client.release();
  }
}

/** Mark unread messages from the other participant as read. Returns affected count. */
export async function markRead(conversationId: string, readerId: string): Promise<number> {
  const { rowCount } = await pool.query(
    `UPDATE chat.messages
       SET read_at = now()
       WHERE conversation_id = $1 AND sender_id <> $2 AND read_at IS NULL`,
    [conversationId, readerId],
  );
  return rowCount ?? 0;
}
