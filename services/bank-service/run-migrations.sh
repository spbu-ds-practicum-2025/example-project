#!/bin/bash
# Migration runner script for Bank Service database

set -e

MIGRATIONS_PATH="./migrations"
DATABASE_URL="${DATABASE_URL:-postgres://bank_service:bank_service@localhost:5433/bank_db?sslmode=disable}"

function show_usage() {
    cat << EOF
Database Migration Tool for Bank Service

Usage:
    ./run-migrations.sh <action> [options]

Actions:
    up          Apply all pending migrations
    down [n]    Rollback migrations (optionally specify number)
    version     Show current migration version
    force <n>   Force set migration version
    create      Create new migration (requires NAME environment variable)

Environment Variables:
    DATABASE_URL    Database connection string (default: postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable)
    NAME            Name for new migration (only with create action)

Examples:
    # Apply all migrations
    ./run-migrations.sh up

    # Rollback last migration
    ./run-migrations.sh down 1

    # Check current version
    ./run-migrations.sh version

    # Create new migration
    NAME="add_user_table" ./run-migrations.sh create

    # Custom database URL
    DATABASE_URL="postgres://user:pass@host:5432/db" ./run-migrations.sh up

EOF
}

function check_migrate() {
    if ! command -v migrate &> /dev/null; then
        echo "Error: 'migrate' command not found"
        echo ""
        echo "Please install golang-migrate:"
        echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        echo ""
        echo "Or download from: https://github.com/golang-migrate/migrate/releases"
        exit 1
    fi
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

# Main script
ACTION=${1:-}

if [ "$ACTION" = "create" ]; then
    create_migration
    exit 0
fi

check_migrate

echo "Bank Service - Database Migration Tool"
echo "======================================="
echo ""

case "$ACTION" in
    up)
        echo "Applying migrations..."
        migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" up
        ;;
    
    down)
        STEPS=${2:-}
        if [ -z "$STEPS" ]; then
            echo "Warning: This will rollback ALL migrations!"
            read -p "Are you sure? (yes/no): " confirm
            if [ "$confirm" != "yes" ]; then
                echo "Cancelled."
                exit 0
            fi
            migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" down
        else
            echo "Rolling back $STEPS migration(s)..."
            migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" down "$STEPS"
        fi
        ;;
    
    version)
        echo "Current migration version:"
        migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" version
        ;;
    
    force)
        VERSION=${2:-}
        if [ -z "$VERSION" ]; then
            echo "Error: Must specify version number"
            exit 1
        fi
        echo "Forcing migration version to $VERSION..."
        migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" force "$VERSION"
        ;;
    
    *)
        show_usage
        exit 1
        ;;
esac

echo ""
echo "Migration completed successfully!"
