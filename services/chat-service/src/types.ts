export interface Conversation {
  id: string;
  listingId: string | null;
  buyerId: string;
  sellerId: string;
  lastMessageAt: string | null;
  createdAt: string;
}

export interface Message {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  clientMsgId: string | null;
  createdAt: string;
  readAt: string | null;
}

export interface ConversationSummary extends Conversation {
  lastMessage: Message | null;
  unread: number;
}

// Socket.io event maps — typed both directions.
export interface MessageNewPayload {
  message: Message;
}

export interface MessageDeliveredPayload {
  conversationId: string;
  messageId: string;
  clientMsgId: string | null;
}

export interface PresenceUpdatePayload {
  userId: string;
  online: boolean;
}

export interface MessageReadPayload {
  conversationId: string;
}

export interface SocketAck {
  ok: boolean;
  error?: string;
  message?: Message;
}

export interface ClientToServerEvents {
  'message:send': (
    payload: { conversationId: string; body: string; clientMsgId: string },
    ack: (res: SocketAck) => void,
  ) => void;
  typing: (payload: { conversationId: string }) => void;
  'message:read': (payload: MessageReadPayload) => void;
}

export interface ServerToClientEvents {
  'message:new': (payload: MessageNewPayload) => void;
  'message:delivered': (payload: MessageDeliveredPayload) => void;
  'presence:update': (payload: PresenceUpdatePayload) => void;
  typing: (payload: { conversationId: string; userId: string }) => void;
  'message:read': (payload: { conversationId: string; userId: string }) => void;
  error: (payload: { error: string }) => void;
}

export interface SocketData {
  userId: string;
  role: string;
}
