import type { Socket } from 'socket.io';
import { verifyAccessToken } from '../lib/jwt';
import type {
  ClientToServerEvents,
  ServerToClientEvents,
  SocketData,
} from '../types';

export type AppSocket = Socket<ClientToServerEvents, ServerToClientEvents, never, SocketData>;

// io.use() handshake middleware: reads handshake.auth.token, verifies RS256 with
// the public key, attaches userId/role. Rejects invalid tokens — an
// unauthenticated socket NEVER reaches a handler or a room.
export function socketAuth(socket: AppSocket, next: (err?: Error) => void): void {
  const token = socket.handshake.auth?.token as string | undefined;
  if (!token) {
    next(new Error('missing token'));
    return;
  }
  try {
    const user = verifyAccessToken(token);
    socket.data.userId = user.userId;
    socket.data.role = user.role;
    next();
  } catch {
    next(new Error('invalid token'));
  }
}
