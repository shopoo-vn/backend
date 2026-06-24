import { MigrationInterface, QueryRunner } from 'typeorm';

// Schema "listing" itself is created by infra/postgres/init-db.sql (the listing_svc
// role only has CREATE-in-schema, not CREATE-schema). This migration creates the
// tables, the full-text search column + GIN index, and seeds default categories.
export class InitListing1719100000000 implements MigrationInterface {
  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`
      CREATE TABLE IF NOT EXISTS listing.categories (
        id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        name       TEXT NOT NULL,
        slug       TEXT NOT NULL UNIQUE,
        parent_id  UUID REFERENCES listing.categories(id) ON DELETE SET NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now()
      );
    `);

    await queryRunner.query(`
      CREATE TABLE IF NOT EXISTS listing.listings (
        id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        seller_id     UUID NOT NULL,
        title         TEXT NOT NULL,
        description   TEXT NOT NULL DEFAULT '',
        price         NUMERIC(14,0) NOT NULL DEFAULT 0,
        category_id   UUID NOT NULL REFERENCES listing.categories(id),
        location      TEXT,
        condition     TEXT NOT NULL DEFAULT 'used'
                      CHECK (condition IN ('new','like_new','used')),
        status        TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','active','rejected','sold','hidden')),
        media_ids     TEXT[] NOT NULL DEFAULT '{}',
        search_vector TSVECTOR GENERATED ALWAYS AS (
          setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
          setweight(to_tsvector('simple', coalesce(description, '')), 'B')
        ) STORED,
        created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
        updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
      );
    `);

    await queryRunner.query(
      `CREATE INDEX IF NOT EXISTS idx_listings_search ON listing.listings USING GIN (search_vector);`,
    );
    await queryRunner.query(
      `CREATE INDEX IF NOT EXISTS idx_listings_category ON listing.listings (category_id);`,
    );
    await queryRunner.query(
      `CREATE INDEX IF NOT EXISTS idx_listings_status_created ON listing.listings (status, created_at DESC);`,
    );
    await queryRunner.query(
      `CREATE INDEX IF NOT EXISTS idx_listings_price ON listing.listings (price);`,
    );

    // Seed top-level electronics categories (idempotent on slug).
    await queryRunner.query(`
      INSERT INTO listing.categories (name, slug) VALUES
        ('Điện thoại', 'dien-thoai'),
        ('Laptop', 'laptop'),
        ('Máy tính bảng', 'may-tinh-bang'),
        ('Máy tính để bàn', 'may-tinh-de-ban'),
        ('TV', 'tv'),
        ('Âm thanh', 'am-thanh'),
        ('Máy ảnh', 'may-anh'),
        ('Phụ kiện', 'phu-kien')
      ON CONFLICT (slug) DO NOTHING;
    `);
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`DROP TABLE IF EXISTS listing.listings;`);
    await queryRunner.query(`DROP TABLE IF EXISTS listing.categories;`);
  }
}
