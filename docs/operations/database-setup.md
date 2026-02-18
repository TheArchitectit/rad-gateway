# RAD Gateway Database Setup Guide

## Overview

RAD Gateway supports both PostgreSQL (recommended for production) and SQLite (fallback/development). This guide covers setting up the database for RAD Gateway.

## PostgreSQL Setup (Recommended)

### Prerequisites

- PostgreSQL 13+ running and accessible
- `psql` client installed
- Superuser access to PostgreSQL

### Quick Setup

1. **Run the automated setup script:**

```bash
# From the repository root
cd /mnt/ollama/git/RADAPI01
./scripts/setup-postgres.sh
```

This will create:
- Database: `radgateway`
- User: `radgateway_user`
- Password: `radgateway_secure_password_2024` (change in production)

2. **Configure RAD Gateway to use PostgreSQL:**

```bash
# Create environment file
sudo mkdir -p /opt/radgateway01/config
sudo tee /opt/radgateway01/config/env << 'EOF'
RAD_DB_DRIVER=postgres
RAD_DB_DSN=postgresql://radgateway_user:radgateway_secure_password_2024@localhost:5432/radgateway?sslmode=disable
RAD_DB_MAX_OPEN_CONNS=10
RAD_DB_MAX_IDLE_CONNS=3
EOF
```

3. **Test the connection:**

```bash
psql "postgresql://radgateway_user:radgateway_secure_password_2024@localhost:5432/radgateway" -c "SELECT 1"
```

### Manual Setup (Alternative)

If you prefer manual setup, run the SQL script:

```bash
# As postgres superuser
psql -U postgres -f /mnt/ollama/git/RADAPI01/scripts/setup-postgres.sql
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RAD_DB_DRIVER` | Database driver: `postgres` or `sqlite` | `sqlite` |
| `RAD_DB_DSN` | Database connection string | `radgateway.db` |
| `RAD_DB_MAX_OPEN_CONNS` | Max open connections (PostgreSQL only) | `10` |
| `RAD_DB_MAX_IDLE_CONNS` | Max idle connections (PostgreSQL only) | `3` |
| `RAD_DB_CONN_MAX_LIFETIME` | Connection max lifetime | `5m` |

## SQLite Setup (Development/Fallback)

SQLite requires no setup. Just use:

```bash
RAD_DB_DRIVER=sqlite
RAD_DB_DSN=radgateway.db
```

The database file will be created automatically.

## Connection Retry Logic

The PostgreSQL implementation includes automatic retry logic:
- **3 attempts** with exponential backoff
- **Initial delay:** 1 second
- **Connection timeout:** 5 seconds per attempt
- **Fallback:** If PostgreSQL fails, automatically falls back to SQLite

## Health Check Integration

The `/health` endpoint includes database status:

```bash
curl http://localhost:8090/health
```

Response:
```json
{
  "status": "ok",
  "database": "ok",
  "driver": "postgres"
}
```

If the database is unavailable:
```json
{
  "status": "ok",
  "database": "degraded",
  "driver": "postgres"
}
```

HTTP status code will be `503 Service Unavailable` if database is unhealthy.

## Migration Behavior

RAD Gateway automatically runs migrations on startup:

1. Connects to database
2. Creates schema if not exists
3. Applies pending migrations
4. Records migration version

Migrations are idempotent - they can safely be run multiple times.

## Troubleshooting

### Connection Refused

```
fatal: connection to server at "localhost", port 5432 failed
```

- Check PostgreSQL is running: `sudo systemctl status postgresql`
- Check connection settings in `/opt/radgateway01/config/env`

### Authentication Failed

```
FATAL: password authentication failed for user "radgateway_user"
```

- Verify password in DSN matches the one set during setup
- Check pg_hba.conf allows the connection method

### Permission Denied

```
ERROR: permission denied for schema public
```

- Run the setup script again to grant proper permissions
- Or manually grant: `GRANT CREATE ON SCHEMA public TO radgateway_user;`

### Fallback to SQLite

If you see this log message:
```
PostgreSQL connection failed: <error>. Falling back to SQLite...
```

This means the PostgreSQL connection failed after 3 retries. RAD Gateway will use SQLite instead. Check:
- PostgreSQL is running
- Network connectivity
- Connection string is correct

## Production Recommendations

1. **Use PostgreSQL** - SQLite is not recommended for production
2. **Enable SSL** - Change `sslmode=disable` to `sslmode=require`
3. **Use strong passwords** - Change the default password
4. **Set resource limits** - Configure `RAD_DB_MAX_OPEN_CONNS` based on your PostgreSQL max_connections
5. **Monitor connections** - Watch for connection pool exhaustion
6. **Backup regularly** - Use PostgreSQL backup tools

## Files Reference

- `/mnt/ollama/git/RADAPI01/scripts/setup-postgres.sh` - Automated setup script
- `/mnt/ollama/git/RADAPI01/scripts/setup-postgres.sql` - SQL setup script
- `/mnt/ollama/git/RADAPI01/config/env.example` - Environment template
- `/mnt/ollama/git/RADAPI01/config/env.local` - Local development config
- `/opt/radgateway01/config/env` - Production configuration
