import type { Request, Response } from 'express';
import * as repo from '../repo/chat.repo';
import { isParticipant } from '../services/chat.service';
import { createConversationSchema, listMessagesQuerySchema } from './validation';
import { logger } from '../lib/logger';

// GET /conversations — the authenticated user's conversations (last msg + unread).
export async function listConversations(req: Request, res: Response): Promise<void> {
  try {
    const items = await repo.listConversations(req.user!.userId);
    res.json({ items });
  } catch (err) {
    logger.error('listConversations failed', { err: (err as Error).message });
    res.status(500).json({ error: 'internal error' });
  }
}

// POST /conversations { sellerId, listingId } — get-or-create. Caller is the buyer.
export async function createConversation(req: Request, res: Response): Promise<void> {
  const parsed = createConversationSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.issues[0]?.message ?? 'invalid body' });
    return;
  }
  const buyerId = req.user!.userId;
  const { sellerId, listingId } = parsed.data;
  if (sellerId === buyerId) {
    res.status(422).json({ error: 'cannot start a conversation with yourself' });
    return;
  }
  try {
    const conv = await repo.getOrCreateConversation(buyerId, sellerId, listingId ?? null);
    res.status(201).json(conv);
  } catch (err) {
    logger.error('createConversation failed', { err: (err as Error).message });
    res.status(500).json({ error: 'internal error' });
  }
}

// GET /conversations/:id/messages?before&limit — paginated history (participant only).
export async function listMessages(req: Request, res: Response): Promise<void> {
  const parsed = listMessagesQuerySchema.safeParse(req.query);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.issues[0]?.message ?? 'invalid query' });
    return;
  }
  try {
    const conv = await repo.findConversationById(req.params.id);
    if (!conv) {
      res.status(404).json({ error: 'conversation not found' });
      return;
    }
    if (!isParticipant(conv, req.user!.userId)) {
      res.status(403).json({ error: 'not a participant' });
      return;
    }
    const messages = await repo.listMessages(
      conv.id,
      parsed.data.before ?? null,
      parsed.data.limit,
    );
    res.json({ items: messages, limit: parsed.data.limit });
  } catch (err) {
    logger.error('listMessages failed', { err: (err as Error).message });
    res.status(500).json({ error: 'internal error' });
  }
}
