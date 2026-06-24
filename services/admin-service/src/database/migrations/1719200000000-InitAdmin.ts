import { MigrationInterface, QueryRunner } from 'typeorm';

// Schema "admin" itself is created by infra/postgres/init-db.sql (the admin_svc
// role only has CREATE-in-schema, not CREATE-schema). This migration creates the
// moderation queue + reports tables.
export class InitAdmin1719200000000 implements MigrationInterface {
  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`
      CREATE TABLE IF NOT EXISTS admin.moderation_items (
        listing_id  UUID PRIMARY KEY,
        seller_id   UUID,
        title       TEXT,
        status      TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','approved','rejected')),
        reviewed_by UUID,
        reason      TEXT,
        created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
        updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
      );
    `);

    await queryRunner.query(`
      CREATE TABLE IF NOT EXISTS admin.reports (
        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        target_type TEXT,
        target_id   UUID,
        reporter_id UUID,
        reason      TEXT,
        status      TEXT NOT NULL DEFAULT 'open',
        created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
      );
    `);

    await queryRunner.query(
      `CREATE INDEX IF NOT EXISTS idx_moderation_items_status ON admin.moderation_items (status, created_at DESC);`,
    );
    await queryRunner.query(
      `CREATE INDEX IF NOT EXISTS idx_reports_status ON admin.reports (status, created_at DESC);`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`DROP TABLE IF EXISTS admin.reports;`);
    await queryRunner.query(`DROP TABLE IF EXISTS admin.moderation_items;`);
  }
}
