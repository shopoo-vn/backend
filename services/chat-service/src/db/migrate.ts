import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { pool } from '../lib/db';
import { config } from '../config';
import { logger } from '../lib/logger';

const MIGRATIONS_DIR = join(__dirname, 'migrations');

/**
 * Small ordered SQL runner. Records applied files in chat.schema_migrations and
 * runs each pending one inside a transaction. Idempotent across restarts.
 */
export async function runMigrations(): Promise<void> {
  const client = await pool.connect();
  try {
    await client.query(`SET search_path TO ${config.db.schema}`);
    await client.query(`
      CREATE TABLE IF NOT EXISTS chat.schema_migrations (
        filename   TEXT PRIMARY KEY,
        applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
      );
    `);

    const files = readdirSync(MIGRATIONS_DIR)
      .filter((f) => f.endsWith('.sql'))
      .sort();

    for (const file of files) {
      const { rowCount } = await client.query(
        'SELECT 1 FROM chat.schema_migrations WHERE filename = $1',
        [file],
      );
      if (rowCount && rowCount > 0) {
        continue;
      }
      const sql = readFileSync(join(MIGRATIONS_DIR, file), 'utf8');
      await client.query('BEGIN');
      try {
        await client.query(sql);
        await client.query('INSERT INTO chat.schema_migrations (filename) VALUES ($1)', [file]);
        await client.query('COMMIT');
        logger.info('migration applied', { file });
      } catch (err) {
        await client.query('ROLLBACK');
        throw err;
      }
    }
  } finally {
    client.release();
  }
}
