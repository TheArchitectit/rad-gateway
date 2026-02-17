# Database Migration Guide for RAD Gateway

## Overview

RAD Gateway uses a robust migration system designed for production safety. This guide covers creating, applying, and rolling back database migrations.

## Safety Features

The migration system includes multiple layers of protection:

1. **Transactional Migrations** - All migrations run in transactions. If a migration fails, changes are automatically rolled back.
2. **Checksum Validation** - Each migration file is checksummed. If a migration is modified after being applied, the system detects it.
3. **Version Tracking** - The `schema_migrations` table tracks exactly which migrations have been applied.
4. **Dry-Run Mode** - Test migrations without actually applying them.
5. **Down Migrations** - Every migration includes a rollback script for safe recovery.

## Migration Naming Convention

Migration files follow this pattern:

```
XXX_description_here.sql
```

Examples:
- `001_create_users.sql`
- `002_add_user_preferences_table.sql`
- `003_create_api_keys.sql`

Rules:
- Use 3-digit version numbers (001, 002, etc.)
- Use underscores between words in the description
- Must have `.sql` extension
- Versions must be unique and sequential

## Migration File Format

Each migration file contains two sections:

```sql
-- Migration: Description of what this does
-- Created at: 2026-02-17T10:00:00Z

-- +migrate Up
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);

-- +migrate Down
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
```

### Guidelines for Writing Migrations

#### Up Migrations (Forward)

- Create tables with `IF NOT EXISTS`
- Add indexes after table creation
- Use transactions for related changes
- Include foreign key constraints
- Test on both SQLite and PostgreSQL (syntax may differ)

#### Down Migrations (Rollback)

- Reverse operations in opposite order of Up
- Use `IF EXISTS` for DROP statements
- Remove indexes before dropping tables
- Be careful with data-loss operations
- Test rollbacks before deploying

## Using the Migration CLI

### Prerequisites

Set the `DATABASE_URL` environment variable:

```bash
# PostgreSQL
export DATABASE_URL="postgres://user:password@localhost/radgateway"

# SQLite
export DATABASE_URL="sqlite3://./radgateway.db"
```

### Commands

#### Check Status

```bash
migrate status
```

Shows:
- Current database version
- Target version (latest available)
- Number of pending migrations
- List of applied and pending migrations

#### Apply Migrations

Apply all pending migrations:

```bash
migrate up
```

Apply specific number of migrations:

```bash
migrate up 3
```

Migrate to specific version:

```bash
migrate up-to 15
```

#### Rollback Migrations

**WARNING: Rollbacks can result in data loss!**

Rollback the last migration:

```bash
migrate down
```

Rollback multiple migrations:

```bash
migrate down 3
```

Rollback to specific version:

```bash
migrate down-to 10
```

#### View Current Version

```bash
migrate version
```

#### Create New Migration

```bash
migrate create "add user preferences table"
```

This creates a file like `004_add_user_preferences_table.sql` in the migrations directory.

#### Verify Migration Integrity

```bash
migrate verify
```

Checks:
- All applied migrations have matching files
- Checksums match recorded values
- No duplicate versions

#### Dry-Run Mode

Test what would happen without making changes:

```bash
migrate up -dry-run
migrate down 2 -dry-run
```

## Production Deployment

### Pre-Deployment Checklist

Before running migrations in production:

- [ ] Test migrations on staging environment
- [ ] Verify down migrations work correctly
- [ ] Create database backup
- [ ] Schedule maintenance window if needed
- [ ] Have rollback plan ready

### Deployment Steps

1. **Backup the database:**
   ```bash
   # PostgreSQL
   pg_dump $DATABASE_URL > backup_$(date +%Y%m%d_%H%M%S).sql

   # SQLite
   cp radgateway.db radgateway_backup_$(date +%Y%m%d_%H%M%S).db
   ```

2. **Check current status:**
   ```bash
   migrate status
   ```

3. **Run migrations:**
   ```bash
   migrate up
   ```

4. **Verify:**
   ```bash
   migrate version
   migrate verify
   ```

### Rollback Procedure

If a migration fails or causes issues:

1. **Assess the situation:**
   ```bash
   migrate status
   ```

2. **If the migration failed during execution:**
   - The transaction automatically rolled back
   - Database is in the previous state
   - Fix the migration and retry

3. **If you need to rollback a successful migration:**
   ```bash
   # Check what would happen
   migrate down -dry-run

   # Actually rollback
   migrate down
   ```

4. **For emergency rollback to specific version:**
   ```bash
   migrate down-to <previous_version>
   ```

## Troubleshooting

### Checksum Mismatch Error

**Problem:** Migration file was modified after being applied.

**Solution:**
1. Do not modify already-applied migrations
2. Create a new migration to make changes
3. If emergency fix is needed, manually update checksum in database (not recommended)

### Missing Migration Error

**Problem:** Database has a migration recorded that doesn't exist in files.

**Solution:**
1. Restore missing migration file from version control
2. Or manually delete record from `schema_migrations` (if intentionally removed)

### Concurrent Migration Error

**Problem:** Multiple processes trying to migrate simultaneously.

**Solution:**
1. Stop all but one migration process
2. Retry the migration
3. Consider using a lock mechanism for CI/CD pipelines

### Migration Timeout

**Problem:** Migration takes longer than the timeout (default 5 minutes).

**Solution:**
1. Increase timeout: `migrate up -timeout 30m`
2. Consider breaking large migrations into smaller chunks
3. Run during low-traffic periods

### Syntax Error

**Problem:** Migration SQL has syntax error.

**Solution:**
1. Fix the SQL in the migration file
2. Re-run the migration
3. The transaction automatically rolls back on error

## Migration Best Practices

### DO

- Write idempotent migrations (can be safely re-run)
- Test on both SQLite and PostgreSQL
- Include down migrations for every up migration
- Use `IF NOT EXISTS` and `IF EXISTS` clauses
- Keep migrations small and focused
- Document complex migrations with comments
- Test rollback scenarios

### DON'T

- Modify already-applied migrations
- Delete migration files without a plan
- Run migrations without backups in production
- Write data migrations that take too long without batching
- Ignore foreign key constraints
- Forget to test the down migration

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Database Migrations

on:
  push:
    branches: [ main ]

jobs:
  migrate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run Migrations (Dry Run)
        run: go run ./cmd/migrate up -dry-run
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}

      - name: Run Migrations
        run: go run ./cmd/migrate up
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}

      - name: Verify Migrations
        run: go run ./cmd/migrate verify
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
```

### Migration Testing in CI

```bash
#!/bin/bash
set -e

# Create test database
export DATABASE_URL="sqlite3://./test_migration.db"

# Run all migrations
migrate up

# Verify
migrate verify

# Test rollback
migrate down

# Roll forward again
migrate up

# Clean up
rm ./test_migration.db

echo "Migration tests passed!"
```

## Schema Evolution Patterns

### Adding a Column

```sql
-- +migrate Up
ALTER TABLE users ADD COLUMN preferences BLOB DEFAULT '{}';

-- +migrate Down
-- Note: SQLite doesn't support DROP COLUMN, recreate table if needed
-- PostgreSQL:
-- ALTER TABLE users DROP COLUMN IF EXISTS preferences;
```

### Creating an Index

```sql
-- +migrate Up
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- +migrate Down
DROP INDEX IF EXISTS idx_users_email;
```

### Adding a Foreign Key

```sql
-- +migrate Up
ALTER TABLE api_keys ADD COLUMN workspace_id TEXT REFERENCES workspaces(id);

-- +migrate Down
-- Note: Requires recreating table to remove FK in SQLite
-- PostgreSQL:
-- ALTER TABLE api_keys DROP COLUMN IF EXISTS workspace_id;
```

### Data Migration

```sql
-- +migrate Up
-- Backfill new column
UPDATE users SET preferences = '{}' WHERE preferences IS NULL;

-- +migrate Down
-- Cannot undo data changes
```

## Emergency Procedures

### Recovering from Failed Production Migration

1. **Stay calm** - The transaction should have rolled back automatically

2. **Check logs:**
   ```bash
   migrate status
   ```

3. **If database is in an inconsistent state:**
   - Stop all application instances
   - Restore from backup
   - Investigate and fix the migration
   - Retry in staging first

### Manual Intervention

If you must manually fix the schema_migrations table:

```sql
-- View current state
SELECT * FROM schema_migrations ORDER BY version;

-- Remove a migration record (DANGEROUS)
DELETE FROM schema_migrations WHERE version = 999;

-- Update checksum (if you know what you're doing)
UPDATE schema_migrations SET checksum = 'new_checksum' WHERE version = 5;
```

**WARNING:** Manual changes to schema_migrations can corrupt your database. Only do this in emergencies.

## Support

For migration issues:

1. Check logs: `migrate status`
2. Review this guide
3. Consult the team lead
4. Escalate to the database administrator

Remember: **When in doubt, back up first!**
