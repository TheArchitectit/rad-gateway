-- PostgreSQL Initialization: Setup Extensions
-- Location: /docker-entrypoint-initdb.d/02-setup-extensions.sql
-- Purpose: Install required PostgreSQL extensions for Infisical and OpenBao
--
-- This script is executed after database creation and sets up extensions
-- that are required by the Golden Stack services.

-- Connect to postgres database first to create extensions
\c postgres

-- ============================================
-- UUID Extension
-- ============================================
-- Provides UUID generation functions
-- Required by: Infisical for entity IDs

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp') THEN
        CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
        RAISE NOTICE 'Created extension: uuid-ossp';
    ELSE
        RAISE NOTICE 'Extension uuid-ossp already exists';
    END IF;
END
$$;

-- ============================================
-- Crypto Extension
-- ============================================
-- Provides cryptographic functions (hashing, encryption)
-- Required by: OpenBao for encryption operations, Infisical for secret hashing

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto') THEN
        CREATE EXTENSION IF NOT EXISTS "pgcrypto";
        RAISE NOTICE 'Created extension: pgcrypto';
    ELSE
        RAISE NOTICE 'Extension pgcrypto already exists';
    END IF;
END
$$;

-- ============================================
-- Infisical Database Extensions
-- ============================================
\c :INFISICAL_DB_NAME

-- UUID extension for Infisical
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp') THEN
        CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
        RAISE NOTICE 'Created extension uuid-ossp in database: %', current_database();
    END IF;
END
$$;

-- Pgcrypto extension for Infisical (for secret encryption/hashing)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto') THEN
        CREATE EXTENSION IF NOT EXISTS "pgcrypto";
        RAISE NOTICE 'Created extension pgcrypto in database: %', current_database();
    END IF;
END
$$;

-- ============================================
-- OpenBao Database Extensions
-- ============================================
\c :OPENBAO_DB_NAME

-- UUID extension for OpenBao
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp') THEN
        CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
        RAISE NOTICE 'Created extension uuid-ossp in database: %', current_database();
    END IF;
END
$$;

-- Pgcrypto extension for OpenBao (for encryption operations)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto') THEN
        CREATE EXTENSION IF NOT EXISTS "pgcrypto";
        RAISE NOTICE 'Created extension pgcrypto in database: %', current_database();
    END IF;
END
$$;

-- ============================================
-- Schema Setup for OpenBao
-- ============================================
-- OpenBao requires specific schema permissions

-- Grant schema permissions to openbao user
GRANT ALL ON SCHEMA public TO :OPENBAO_DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO :OPENBAO_DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO :OPENBAO_DB_USER;

-- ============================================
-- Schema Setup for Infisical
-- ============================================
\c :INFISICAL_DB_NAME

-- Grant schema permissions to infisical user
GRANT ALL ON SCHEMA public TO :INFISICAL_DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO :INFISICAL_DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO :INFISICAL_DB_USER;

-- ============================================
-- Verification
-- ============================================
\c postgres

SELECT
    current_database() as database,
    extname as extension_name,
    extversion as version
FROM pg_extension
WHERE extname IN ('uuid-ossp', 'pgcrypto')
ORDER BY extname;

-- Log completion
DO $$
BEGIN
    RAISE NOTICE 'PostgreSQL extensions setup completed successfully';
END
$$;
