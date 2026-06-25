import { z } from 'zod';
import type { Server } from 'socket.io';
import type { AppSocket } from './auth';
import type {
  ClientToServerEvents,
  ServerToClientEvents,
  SocketData,
} from '../types';
import * as repo from '../repo/chat.repo';
import { isParticipant, otherParticipant, persistMessage } from '../services/chat.service';
import { addSocket, removeSocket } from '../services/presence.service';
import { logger } from '../lib/logger';

export type AppServer = Server<ClientToServerEvents, ServerToClientEvents, never, SocketData>;

const room = (conversationId: string): string => `conv:${conversationId}`;

const messageSendSchema = z.object({
  conversationId: z.string().uuid(),
  body: z.string().trim().min(1).max(4000),
  clientMsgId: z.string().min(1).max(128),
});
const conversationOnlySchema = z.object({ conversationId: z.string().uuid() });

// Notify a peer of this user's presence change, if the peer has a socket.
async function broadcastPresence(io: AppServer, userId: string, online: boolean): Promise<void> {
  // Find conversations and emit to the OTHER participant's user-room.
  const convs = await repo.listConversations(userId);
  const peers = new Set<string>();
  for (const c of convs) {
    peers.add(otherParticipant(c, userId));
  }
  for (const peer of peers) {
    io.to(`user:${peer}`).emit('presence:update', { userId, online });
  }
}

export function registerHandlers(io: AppServer, socket: AppSocket): void {
  const { userId } = socket.data;

  // Personal room so peers can target presence/notifications at this user.
  void socket.join(`user:${userId}`);

  void addSocket(userId, socket.id)
    .then((cameOnline) => {
      if (cameOnline) {
        void broadcastPresence(io, userId, true);
      }
    })
    .catch((err) => logger.error('presence add failed', { err: (err as Error).message, userId }));

  // message:send — validate, authorize, PERSIST first, then emit. Acked.
  socket.on('message:send', (payload, ack) => {
    // Clients (e.g. mobile) may emit without an ack callback — never crash on it.
    const safeAck = typeof ack === 'function' ? ack : () => undefined;
    void (async () => {
      const parsed = messageSendSchema.safeParse(payload);
      if (!parsed.success) {
        safeAck({ ok: false, error: parsed.error.issues[0]?.message ?? 'invalid payload' });
        return;
      }
      const { conversationId, body, clientMsgId } = parsed.data;
      try {
        const conv = await repo.findConversationById(conversationId);
        if (!conv) {
          safeAck({ ok: false, error: 'conversation not found' });
          return;
        }
        if (!isParticipant(conv, userId)) {
          safeAck({ ok: false, error: 'not a participant' });
          return;
        }

        // senderId comes from the verified socket, NEVER the client payload.
        const { message } = await persistMessage({
          conversation: conv,
          senderId: userId,
          body,
          clientMsgId,
        });

        // Ensure sender is in the room, then fan out.
        await socket.join(room(conversationId));
        io.to(room(conversationId)).emit('message:new', { message });
        safeAck({ ok: true, message });

        // Tell the sender the server accepted/persisted it.
        socket.emit('message:delivered', {
          conversationId,
          messageId: message.id,
          clientMsgId: message.clientMsgId,
        });
      } catch (err) {
        logger.error('message:send failed', { err: (err as Error).message, userId });
        safeAck({ ok: false, error: 'internal error' });
      }
    })();
  });

  // typing — relay to the room (no persistence).
  socket.on('typing', (payload) => {
    void (async () => {
      const parsed = conversationOnlySchema.safeParse(payload);
      if (!parsed.success) {
        return;
      }
      const conv = await repo.findConversationById(parsed.data.conversationId);
      if (!conv || !isParticipant(conv, userId)) {
        return;
      }
      await socket.join(room(conv.id));
      socket.to(room(conv.id)).emit('typing', { conversationId: conv.id, userId });
    })();
  });

  // message:read — mark peer's messages read, notify the room.
  socket.on('message:read', (payload) => {
    void (async () => {
      const parsed = conversationOnlySchema.safeParse(payload);
      if (!parsed.success) {
        return;
      }
      const conv = await repo.findConversationById(parsed.data.conversationId);
      if (!conv || !isParticipant(conv, userId)) {
        return;
      }
      try {
        await repo.markRead(conv.id, userId);
        await socket.join(room(conv.id));
        io.to(room(conv.id)).emit('message:read', { conversationId: conv.id, userId });
      } catch (err) {
        logger.error('message:read failed', { err: (err as Error).message, userId });
      }
    })();
  });

  socket.on('disconnect', () => {
    void removeSocket(userId, socket.id)
      .then((wentOffline) => {
        if (wentOffline) {
          void broadcastPresence(io, userId, false);
        }
      })
      .catch((err) =>
        logger.error('presence remove failed', { err: (err as Error).message, userId }),
      );
  });
}
