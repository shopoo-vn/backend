import express, { type Express, type NextFunction, type Request, type Response } from 'express';
import { config } from './config';
import { router } from './http/routes';
import { logger } from './lib/logger';

export function createApp(): Express {
  const app = express();

  app.use(express.json({ limit: '64kb' }));

  // CORS allow-list (no "*" with credentials per security baseline).
  app.use((req: Request, res: Response, next: NextFunction) => {
    const origin = req.header('origin');
    const { origins } = config.cors;
    if (origins.includes('*')) {
      res.header('Access-Control-Allow-Origin', '*');
    } else if (origin && origins.includes(origin)) {
      res.header('Access-Control-Allow-Origin', origin);
      res.header('Vary', 'Origin');
    }
    res.header('Access-Control-Allow-Methods', 'GET,POST,OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Authorization,Content-Type');
    if (req.method === 'OPTIONS') {
      res.sendStatus(204);
      return;
    }
    next();
  });

  // Health — no auth, liveness only.
  app.get('/health', (_req, res) => {
    res.json({ status: 'ok' });
  });

  app.use(router);

  app.use((_req, res) => {
    res.status(404).json({ error: 'not found' });
  });

  // Final error handler — never leak stack traces.
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  app.use((err: Error, _req: Request, res: Response, _next: NextFunction) => {
    logger.error('unhandled error', { err: err.message });
    if (res.headersSent) {
      return;
    }
    res.status(500).json({ error: 'internal error' });
  });

  return app;
}
