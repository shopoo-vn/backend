import type { Conversation, Message } from '../types';
import * as repo from '../repo/chat.repo';
import { publishEvent } from '../lib/rabbitmq';

export function isParticipant(conv: Conversation, userId: string): boolean {
  return conv.buyerId === userId || conv.sellerId === userId;
}

export function otherParticipant(conv: Conversation, userId: string): string {
  return conv.buyerId === userId ? conv.sellerId : conv.buyerId;
}

/**
 * Persist a message FIRST (idempotent on clientMsgId), then publish
 * chat.message.created AFTER the DB commit. Emitting over sockets is the
 * caller's job (so it stays out of the service that owns persistence).
 */
export async function persistMessage(params: {
  conversation: Conversation;
  senderId: string;
  body: string;
  clientMsgId: string | null;
}): Promise<{ message: Message; created: boolean }> {
  const { conversation, senderId, body, clientMsgId } = params;
  const result = await repo.insertMessage(conversation.id, senderId, body, clientMsgId);

  // Only publish on a genuinely new message (avoid duplicate notifications on retry).
  if (result.created) {
    publishEvent('chat.message.created', {
      conversationId: conversation.id,
      messageId: result.message.id,
      senderId,
      recipientId: otherParticipant(conversation, senderId),
    });
  }
  return result;
}
