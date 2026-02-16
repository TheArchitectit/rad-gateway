#!/bin/bash
#
# PostgreSQL Initialization Script: Create Databases and Users
# Location: /docker-entrypoint-initdb.d/01-create-databases.sh
# Purpose: Create Infisical and OpenBao databases with proper permissions
#

set -euo pipefail

# Logging functions
log_info() {
    echo "[INIT] [$(date -Iseconds)] INFO: $*"
}

log_error() {
    echo "[INIT] [$(date -Iseconds)] ERROR: $*" >&2
}

log_warn() {
    echo "[INIT] [$(date -Iseconds)] WARN: $*"
}

# Function to read password from file or environment variable
get_password() {
    local password_file="$1"
    local env_var="$2"

    if [[ -f "$password_file" ]]; then
        cat "$password_file" | tr -d '[:space:]'
    elif [[ -n "${!env_var:-}" ]]; then
        echo "${!env_var}"
    else
        log_error "Password not found in file: $password_file or env: $env_var"
        return 1
    fi
}

# Function to create database and user
create_database_and_user() {
    local db_name="$1"
    local db_user="$2"
    local db_password="$3"
    local description="$4"

    log_info "Creating $description database and user..."

    # Check if user already exists
    local user_exists
    user_exists=$(psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$db_user'" 2>/dev/null || echo "0")

    if [[ "$user_exists" == "1" ]]; then
        log_warn "User '$db_user' already exists, updating password..."
        psql -v ON_ERROR_STOP=1 -c "ALTER USER \"$db_user\" WITH PASSWORD '$db_password';" 2>/dev/null || {
            log_error "Failed to update password for user '$db_user'"
            return 1
        }
    else
        # Create user with password
        psql -v ON_ERROR_STOP=1 -c "CREATE USER \"$db_user\" WITH PASSWORD '$db_password';" 2>/dev/null || {
            log_error "Failed to create user '$db_user'"
            return 1
        }
        log_info "Created user '$db_user'"
    fi

    # Check if database already exists
    local db_exists
    db_exists=$(psql -tAc "SELECT 1 FROM pg_database WHERE datname='$db_name'" 2>/dev/null || echo "0")

    if [[ "$db_exists" == "1" ]]; then
        log_warn "Database '$db_name' already exists"
    else
        # Create database with owner
        psql -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"$db_name\" OWNER \"$db_user\" ENCODING 'UTF8' LC_COLLATE 'en_US.UTF-8' LC_CTYPE 'en_US.UTF-8' TEMPLATE template0;" 2>/dev/null || {
            log_error "Failed to create database '$db_name'"
            return 1
        }
        log_info "Created database '$db_name'"
    fi

    # Grant all privileges on database to user
    psql -v ON_ERROR_STOP=1 -c "GRANT ALL PRIVILEGES ON DATABASE \"$db_name\" TO \"$db_user\";" 2>/dev/null || {
        log_error "Failed to grant privileges on database '$db_name'"
        return 1
    }

    # Grant schema permissions
    psql -d "$db_name" -v ON_ERROR_STOP=1 -c "
        GRANT ALL ON SCHEMA public TO \"$db_user\";
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO \"$db_user\";
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO \"$db_user\";
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO \"$db_user\";
    " 2>/dev/null || {
        log_warn "Failed to grant schema privileges on '$db_name' (may already exist)"
    }

    log_info "Granted privileges on database '$db_name' to user '$db_user'"

    return 0
}

# Main initialization
main() {
    log_info "Starting PostgreSQL database initialization..."

    # Get passwords
    local infisical_password
    local openbao_password

    infisical_password=$(get_password "${INFISICAL_DB_PASSWORD_FILE:-}" "INFISICAL_DB_PASSWORD") || {
        log_error "Failed to get Infisical database password"
        exit 1
    }

    openbao_password=$(get_password "${OPENBAO_DB_PASSWORD_FILE:-}" "OPENBAO_DB_PASSWORD") || {
        log_error "Failed to get OpenBao database password"
        exit 1
    }

    # Validate passwords are not empty
    if [[ -z "$infisical_password" ]]; then
        log_error "Infisical database password is empty"
        exit 1
    fi

    if [[ -z "$openbao_password" ]]; then
        log_error "OpenBao database password is empty"
        exit 1
    fi

    log_info "Passwords loaded successfully"

    # Get database and user names from environment or use defaults
    local infisical_db="${INFISICAL_DB_NAME:-infisical}"
    local infisical_user="${INFISICAL_DB_USER:-infisical}"
    local openbao_db="${OPENBAO_DB_NAME:-openbao}"
    local openbao_user="${OPENBAO_DB_USER:-openbao}"

    # Create Infisical database and user
    create_database_and_user \
        "$infisical_db" \
        "$infisical_user" \
        "$infisical_password" \
        "Infisical" || {
        log_error "Failed to create Infisical database and user"
        exit 1
    }

    # Create OpenBao database and user
    create_database_and_user \
        "$openbao_db" \
        "$openbao_user" \
        "$openbao_password" \
        "OpenBao" || {
        log_error "Failed to create OpenBao database and user"
        exit 1
    }

    # Verify databases were created
    log_info "Verifying database creation..."

    local infisical_exists
    local openbao_exists

    infisical_exists=$(psql -tAc "SELECT 1 FROM pg_database WHERE datname='$infisical_db'")
    openbao_exists=$(psql -tAc "SELECT 1 FROM pg_database WHERE datname='$openbao_db'")

    if [[ "$infisical_exists" != "1" ]]; then
        log_error "Verification failed: Infisical database does not exist"
        exit 1
    fi

    if [[ "$openbao_exists" != "1" ]]; then
        log_error "Verification failed: OpenBao database does not exist"
        exit 1
    fi

    log_info "All databases created and verified successfully"
    log_info "Infisical database: $infisical_db (user: $infisical_user)"
    log_info "OpenBao database: $openbao_db (user: $openbao_user)"

    return 0
}

# Execute main function
main "$@"
