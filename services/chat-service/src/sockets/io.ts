import type { Server as HttpServer } from 'node:http';
import { Server } from 'socket.io';
import { createAdapter } from '@socket.io/redis-adapter';
import { config } from '../config';
import { pubClient, subClient } from '../lib/redis';
import { logger } from '../lib/logger';
import { socketAuth, type AppSocket } from './auth';
import { registerHandlers, type AppServer } from './handlers';

export function createIo(httpServer: HttpServer): AppServer {
  const io: AppServer = new Server(httpServer, {
    cors: {
      origin: config.cors.origins.includes('*') ? true : config.cors.origins,
      methods: ['GET', 'POST'],
    },
  });

  // Redis adapter so emits reach sockets on other instances (horizontal scale).
  io.adapter(createAdapter(pubClient, subClient));

  io.use(socketAuth);

  io.on('connection', (socket: AppSocket) => {
    logger.info('socket connected', { userId: socket.data.userId, socketId: socket.id });
    registerHandlers(io, socket);
  });

  return io;
}
