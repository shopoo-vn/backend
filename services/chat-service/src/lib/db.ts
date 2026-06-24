import { Pool } from 'pg';
import { config } from '../config';
import { logger } from './logger';

// Single shared connection pool. search_path is pinned to the service schema so
// unqualified identifiers resolve to "chat".
export const pool = new Pool({
  connectionString: config.db.url,
  max: 10,
});

pool.on('connect', (client) => {
  void client.query(`SET search_path TO ${config.db.schema}`);
});

pool.on('error', (err) => {
  logger.error('unexpected idle pg client error', { err: err.message });
});

export async function closeDb(): Promise<void> {
  await pool.end();
}
