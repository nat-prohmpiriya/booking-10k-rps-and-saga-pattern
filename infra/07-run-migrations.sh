#!/bin/bash
set -e

# Load environment
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/.env"

# Use values from .env
PG_POD="booking-rush-pg-postgresql-0"
MIGRATIONS_DIR="../scripts/migrations"

echo "Using config from .env:"
echo "  HOST: $HOST"
echo "  NAMESPACE: $NAMESPACE"
echo "  PG_DATABASE: $PG_DATABASE"
echo ""

echo "=== Running Database Migrations ==="

# Get list of .up.sql files sorted by name
MIGRATION_FILES=$(ls -1 "$SCRIPT_DIR/$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | sort)

if [ -z "$MIGRATION_FILES" ]; then
    echo "No migration files found in $MIGRATIONS_DIR"
    exit 1
fi

# Create schema_migrations table if not exists
echo "Creating schema_migrations table..."
ssh ${SSH_USER}@${HOST} "kubectl exec $PG_POD -n $NAMESPACE -- bash -c 'PGPASSWORD=${PG_PASSWORD} psql -U ${PG_USERNAME} -d ${PG_DATABASE} -c \"
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
\"'"

# Get already applied migrations
echo "Checking applied migrations..."
APPLIED=$(ssh ${SSH_USER}@${HOST} "kubectl exec $PG_POD -n $NAMESPACE -- bash -c 'PGPASSWORD=${PG_PASSWORD} psql -U ${PG_USERNAME} -d ${PG_DATABASE} -t -c \"SELECT version FROM schema_migrations;\"'" | tr -d ' ')

# Run each migration
for FILE in $MIGRATION_FILES; do
    FILENAME=$(basename "$FILE")
    VERSION="${FILENAME%.up.sql}"

    # Check if already applied
    if echo "$APPLIED" | grep -q "^${VERSION}$"; then
        echo "  [SKIP] $FILENAME (already applied)"
        continue
    fi

    echo "  [RUN]  $FILENAME"

    # Read SQL content
    SQL_CONTENT=$(cat "$FILE")

    # Copy migration file to server and run
    scp "$FILE" ${SSH_USER}@${HOST}:/tmp/migration.sql > /dev/null

    ssh ${SSH_USER}@${HOST} "kubectl cp /tmp/migration.sql $NAMESPACE/$PG_POD:/tmp/migration.sql"

    ssh ${SSH_USER}@${HOST} "kubectl exec $PG_POD -n $NAMESPACE -- bash -c 'PGPASSWORD=${PG_PASSWORD} psql -U ${PG_USERNAME} -d ${PG_DATABASE} -f /tmp/migration.sql'"

    # Record migration
    ssh ${SSH_USER}@${HOST} "kubectl exec $PG_POD -n $NAMESPACE -- bash -c 'PGPASSWORD=${PG_PASSWORD} psql -U ${PG_USERNAME} -d ${PG_DATABASE} -c \"INSERT INTO schema_migrations (version) VALUES ('\\''$VERSION'\\'');\"'"

    echo "  [DONE] $FILENAME"
done

echo ""
echo "=== Migrations Complete ==="

# Show tables
echo ""
echo "Database tables:"
ssh ${SSH_USER}@${HOST} "kubectl exec $PG_POD -n $NAMESPACE -- bash -c 'PGPASSWORD=${PG_PASSWORD} psql -U ${PG_USERNAME} -d ${PG_DATABASE} -c \"\\dt\"'"
