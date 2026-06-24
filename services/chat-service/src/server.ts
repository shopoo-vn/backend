import { createServer } from 'node:http';
import { config } from './config';
import { logger } from './lib/logger';
import { createApp } from './app';
import { createIo } from './sockets/io';
import { runMigrations } from './db/migrate';
import { initRabbit, closeRabbit } from './lib/rabbitmq';
import { closeDb } from './lib/db';
import { closeRedis } from './lib/redis';
import type { AppServer } from './sockets/handlers';

async function main(): Promise<void> {
  // Run migrations first — fail-fast if the DB/schema isn't ready.
  await runMigrations();
  await initRabbit();

  const app = createApp();
  const httpServer = createServer(app);
  const io: AppServer = createIo(httpServer);

  await new Promise<void>((resolve) => {
    httpServer.listen(config.http.port, resolve);
  });
  logger.info('chat-service listening', { port: config.http.port });

  let shuttingDown = false;
  const shutdown = async (signal: string): Promise<void> => {
    if (shuttingDown) {
      return;
    }
    shuttingDown = true;
    logger.info('shutting down', { signal });

    // Stop accepting new sockets/connections, then drain.
    io.close();
    await new Promise<void>((resolve) => httpServer.close(() => resolve()));

    await closeRabbit();
    await closeRedis();
    await closeDb();
    logger.info('shutdown complete');
    process.exit(0);
  };

  process.on('SIGTERM', () => void shutdown('SIGTERM'));
  process.on('SIGINT', () => void shutdown('SIGINT'));
}

main().catch((err) => {
  logger.error('fatal startup error', { err: (err as Error).message });
  process.exit(1);
});
