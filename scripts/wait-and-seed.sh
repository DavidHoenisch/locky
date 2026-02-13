#!/bin/sh
# Wait for database and run seed SQL

set -e

DB_HOST="${DB_HOST:-postgres}"
DB_USER="${DB_USER:-locky}"
DB_NAME="${DB_NAME:-locky}"
DB_PASSWORD="${DB_PASSWORD:-lockysecret}"
ADMIN_API_KEY="${ADMIN_API_KEY:-test-admin-key-123}"

echo "Waiting for PostgreSQL to be ready..."

# Wait for PostgreSQL to be ready
until PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -U "${DB_USER}" -d "${DB_NAME}" -c '\q' 2>/dev/null; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 1
done

echo "PostgreSQL is ready! Running seed script..."

# Run the seed SQL
PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -U "${DB_USER}" -d "${DB_NAME}" -f /seed.sql

echo ""
echo "=========================================="
echo "Seed data created successfully!"
echo "=========================================="
echo ""
echo "Test Credentials:"
echo "  Tenant: test"
echo "  User: test@example.com"
echo "  Password: password123"
echo ""
echo "OAuth Client:"
echo "  Client ID: test-client-id"
echo "  Client Secret: (none - public client with PKCE)"
echo "  Redirect URI: http://localhost:3000/callback"
echo ""
echo "Admin API Key: ${ADMIN_API_KEY}"
echo "=========================================="
