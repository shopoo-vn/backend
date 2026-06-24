-- Runs once on first Postgres init (as POSTGRES_USER on POSTGRES_DB).
-- Creates one schema + one least-privilege login role per service.
-- DB-per-service is emulated via schema-per-service on a single instance.

-- ── Schemas ────────────────────────────────────────────────────────────────
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS media;
CREATE SCHEMA IF NOT EXISTS noti;
CREATE SCHEMA IF NOT EXISTS listing;
CREATE SCHEMA IF NOT EXISTS admin;
CREATE SCHEMA IF NOT EXISTS chat;

-- ── Per-service roles ────────────────────────────────────────────────────────
-- NOTE: dev-only passwords. Override before any real deployment.
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'auth_svc')    THEN CREATE ROLE auth_svc    LOGIN PASSWORD 'auth_pw';    END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'media_svc')   THEN CREATE ROLE media_svc   LOGIN PASSWORD 'media_pw';   END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'noti_svc')    THEN CREATE ROLE noti_svc    LOGIN PASSWORD 'noti_pw';    END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'listing_svc') THEN CREATE ROLE listing_svc LOGIN PASSWORD 'listing_pw'; END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'admin_svc')   THEN CREATE ROLE admin_svc   LOGIN PASSWORD 'admin_pw';   END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'chat_svc')    THEN CREATE ROLE chat_svc    LOGIN PASSWORD 'chat_pw';    END IF;
END
$$;

-- ── Grants: each role owns/controls only its own schema ──────────────────────
GRANT USAGE, CREATE ON SCHEMA auth    TO auth_svc;
GRANT USAGE, CREATE ON SCHEMA media   TO media_svc;
GRANT USAGE, CREATE ON SCHEMA noti    TO noti_svc;
GRANT USAGE, CREATE ON SCHEMA listing TO listing_svc;
GRANT USAGE, CREATE ON SCHEMA admin   TO admin_svc;
GRANT USAGE, CREATE ON SCHEMA chat    TO chat_svc;

-- Default search_path so each service resolves unqualified tables to its schema.
ALTER ROLE auth_svc    SET search_path = auth;
ALTER ROLE media_svc   SET search_path = media;
ALTER ROLE noti_svc    SET search_path = noti;
ALTER ROLE listing_svc SET search_path = listing;
ALTER ROLE admin_svc   SET search_path = admin;
ALTER ROLE chat_svc    SET search_path = chat;
