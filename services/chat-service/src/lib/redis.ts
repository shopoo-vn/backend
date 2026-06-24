import Redis from 'ioredis';
import { config } from '../config';
import { logger } from './logger';

// Three clients: one for app commands (presence), two dedicated pub/sub
// connections required by @socket.io/redis-adapter.
export const redis = new Redis(config.redis.url, { lazyConnect: false });
export const pubClient = new Redis(config.redis.url, { lazyConnect: false });
export const subClient = pubClient.duplicate();

for (const [name, client] of [
  ['redis', redis],
  ['pub', pubClient],
  ['sub', subClient],
] as const) {
  client.on('error', (err: Error) => logger.error(`redis ${name} error`, { err: err.message }));
}

export async function closeRedis(): Promise<void> {
  await Promise.allSettled([redis.quit(), pubClient.quit(), subClient.quit()]);
}
