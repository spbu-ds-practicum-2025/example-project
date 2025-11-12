#!/bin/bash
# Migration runner script for Analytics Service ClickHouse database

set -e

MIGRATIONS_PATH="./migrations"
CLICKHOUSE_HOST="${CLICKHOUSE_HOST:-localhost}"
CLICKHOUSE_PORT="${CLICKHOUSE_PORT:-9000}"
CLICKHOUSE_DB="${CLICKHOUSE_DB:-analytics}"
CLICKHOUSE_USER="${CLICKHOUSE_USER:-default}"
CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-}"

function show_usage() {
    cat << EOF
Database Migration Tool for Analytics Service (ClickHouse)

Usage:
    ./run-migrations.sh <action> [options]

Actions:
    up          Apply all pending migrations
    down [n]    Rollback migrations (optionally specify number)
    create      Create new migration (requires NAME environment variable)
    status      Show migration status

Environment Variables:
    CLICKHOUSE_HOST      ClickHouse host (default: localhost)
    CLICKHOUSE_PORT      ClickHouse port (default: 9000)
    CLICKHOUSE_DB        Database name (default: analytics)
    CLICKHOUSE_USER      Username (default: default)
    CLICKHOUSE_PASSWORD  Password (default: empty)
    NAME                 Name for new migration (only with create action)

Examples:
    # Apply all migrations
    ./run-migrations.sh up

    # Rollback last migration
    ./run-migrations.sh down 1

    # Show migration status
    ./run-migrations.sh status

    # Create new migration
    NAME="add_topup_operations" ./run-migrations.sh create

    # Custom ClickHouse connection
    CLICKHOUSE_HOST="clickhouse.example.com" CLICKHOUSE_DB="analytics_prod" ./run-migrations.sh up

EOF
}

function check_clickhouse_client() {
    if ! command -v clickhouse-client &> /dev/null; then
        echo "Error: 'clickhouse-client' command not found"
        echo ""
        echo "Please install ClickHouse client:"
        echo "  - Ubuntu/Debian: sudo apt-get install clickhouse-client"
        echo "  - macOS: brew install clickhouse"
        echo "  - Or download from: https://clickhouse.com/docs/en/install"
        exit 1
    fi
}

function get_clickhouse_cmd() {
    local cmd="clickhouse-client --host $CLICKHOUSE_HOST --port $CLICKHOUSE_PORT"
    
    if [ -n "$CLICKHOUSE_USER" ]; then
        cmd="$cmd --user $CLICKHOUSE_USER"
    fi
    
    if [ -n "$CLICKHOUSE_PASSWORD" ]; then
        cmd="$cmd --password $CLICKHOUSE_PASSWORD"
    fi
    
    if [ -n "$CLICKHOUSE_DB" ]; then
        cmd="$cmd --database $CLICKHOUSE_DB"
    fi
    
    echo "$cmd"
}

function create_migrations_table() {
    local cmd=$(get_clickhouse_cmd)
    
    echo "Creating migrations tracking table..."
    $cmd --query "CREATE TABLE IF NOT EXISTS schema_migrations (
        version UInt32,
        dirty UInt8,
        applied_at DateTime DEFAULT now()
    ) ENGINE = MergeTree()
    ORDER BY version
    PRIMARY KEY version"
}

function get_current_version() {
    local cmd=$(get_clickhouse_cmd)
    $cmd --query "SELECT max(version) FROM schema_migrations WHERE dirty = 0" 2>/dev/null || echo "0"
}

function get_next_migration_number() {
    local last_file=$(ls -1 "$MIGRATIONS_PATH"/*.up.sql 2>/dev/null | sort | tail -1)
    if [ -z "$last_file" ]; then
        echo "001"
        return
    fi
    
    local last_number=$(basename "$last_file" | sed 's/_.*//')
    local next_number=$((10#$last_number + 1))
    printf "%03d" "$next_number"
}

function create_migration() {
    if [ -z "$NAME" ]; then
        echo "Error: NAME environment variable is required"
        echo "Usage: NAME='your_migration_name' ./run-migrations.sh create"
        exit 1
    fi
    
    local number=$(get_next_migration_number)
    local safe_name=$(echo "$NAME" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_]/_/g')
    
    local up_file="$MIGRATIONS_PATH/${number}_${safe_name}.up.sql"
    local down_file="$MIGRATIONS_PATH/${number}_${safe_name}.down.sql"
    
    cat > "$up_file" << EOF
-- Migration: $NAME
-- Created: $(date '+%Y-%m-%d %H:%M:%S')

-- Add your migration SQL here

EOF

    cat > "$down_file" << EOF
-- Rollback: $NAME

-- Add your rollback SQL here

EOF

    echo "Created migration files:"
    echo "  - $up_file"
    echo "  - $down_file"
}

function apply_migration() {
    local version=$1
    local file=$2
    local cmd=$(get_clickhouse_cmd)
    
    echo "Applying migration $version: $(basename $file)"
    
    # Mark as dirty
    $cmd --query "INSERT INTO schema_migrations (version, dirty) VALUES ($version, 1)"
    
    # Apply migration
    $cmd < "$file"
    
    # Mark as clean
    $cmd --query "ALTER TABLE schema_migrations DELETE WHERE version = $version"
    $cmd --query "INSERT INTO schema_migrations (version, dirty) VALUES ($version, 0)"
    
    echo "  ✓ Applied"
}

function rollback_migration() {
    local version=$1
    local file=$2
    local cmd=$(get_clickhouse_cmd)
    
    echo "Rolling back migration $version: $(basename $file)"
    
    # Mark as dirty
    $cmd --query "INSERT INTO schema_migrations (version, dirty) VALUES ($version, 1)"
    
    # Rollback migration
    $cmd < "$file"
    
    # Remove from tracking
    $cmd --query "ALTER TABLE schema_migrations DELETE WHERE version = $version"
    
    echo "  ✓ Rolled back"
}

function migrate_up() {
    create_migrations_table
    
    local current_version=$(get_current_version)
    echo "Current version: $current_version"
    
    local applied=0
    
    for up_file in $(ls -1 "$MIGRATIONS_PATH"/*.up.sql 2>/dev/null | sort); do
        local version=$(basename "$up_file" | sed 's/_.*//' | sed 's/^0*//')
        
        if [ "$version" -gt "$current_version" ]; then
            apply_migration "$version" "$up_file"
            applied=$((applied + 1))
        fi
    done
    
    if [ $applied -eq 0 ]; then
        echo "No pending migrations to apply"
    else
        echo "Applied $applied migration(s)"
    fi
}

function migrate_down() {
    local steps=${1:-1}
    
    create_migrations_table
    
    local current_version=$(get_current_version)
    echo "Current version: $current_version"
    
    if [ "$current_version" -eq 0 ]; then
        echo "No migrations to rollback"
        return
    fi
    
    local rolled_back=0
    
    # Get migrations in reverse order
    for down_file in $(ls -1r "$MIGRATIONS_PATH"/*.down.sql 2>/dev/null); do
        if [ $rolled_back -ge $steps ]; then
            break
        fi
        
        local version=$(basename "$down_file" | sed 's/_.*//' | sed 's/^0*//')
        
        if [ "$version" -le "$current_version" ]; then
            rollback_migration "$version" "$down_file"
            rolled_back=$((rolled_back + 1))
            current_version=$((current_version - 1))
        fi
    done
    
    echo "Rolled back $rolled_back migration(s)"
}

function show_status() {
    create_migrations_table
    
    local cmd=$(get_clickhouse_cmd)
    local current_version=$(get_current_version)
    
    echo "Migration Status"
    echo "================"
    echo ""
    echo "Current version: $current_version"
    echo ""
    echo "Available migrations:"
    
    for up_file in $(ls -1 "$MIGRATIONS_PATH"/*.up.sql 2>/dev/null | sort); do
        local version=$(basename "$up_file" | sed 's/_.*//' | sed 's/^0*//')
        local name=$(basename "$up_file" .up.sql | sed 's/^[0-9]*_//')
        
        if [ "$version" -le "$current_version" ]; then
            echo "  ✓ $version - $name (applied)"
        else
            echo "  ○ $version - $name (pending)"
        fi
    done
}

# Main script
ACTION=${1:-}

if [ "$ACTION" = "create" ]; then
    create_migration
    exit 0
fi

check_clickhouse_client

echo "Analytics Service - ClickHouse Migration Tool"
echo "=============================================="
echo ""

case "$ACTION" in
    up)
        echo "Applying migrations..."
        migrate_up
        ;;
    
    down)
        STEPS=${2:-1}
        echo "Rolling back $STEPS migration(s)..."
        migrate_down "$STEPS"
        ;;
    
    status)
        show_status
        ;;
    
    *)
        show_usage
        exit 1
        ;;
esac

echo ""
echo "Migration completed successfully!"
