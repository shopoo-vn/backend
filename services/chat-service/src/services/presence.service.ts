import { redis } from '../lib/redis';

// Presence in Redis: online:<userId> -> set of socketIds. A user may hold
// multiple sockets; only mark offline when the last one disconnects.
const key = (userId: string): string => `online:${userId}`;

/** Add a socket; returns true if this is the user's FIRST socket (came online). */
export async function addSocket(userId: string, socketId: string): Promise<boolean> {
  const added = await redis.sadd(key(userId), socketId);
  const total = await redis.scard(key(userId));
  return added === 1 && total === 1;
}

/** Remove a socket; returns true if it was the user's LAST socket (went offline). */
export async function removeSocket(userId: string, socketId: string): Promise<boolean> {
  await redis.srem(key(userId), socketId);
  const remaining = await redis.scard(key(userId));
  if (remaining === 0) {
    await redis.del(key(userId));
    return true;
  }
  return false;
}

export async function isOnline(userId: string): Promise<boolean> {
  return (await redis.scard(key(userId))) > 0;
}
