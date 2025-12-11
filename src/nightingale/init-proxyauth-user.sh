#!/bin/sh
# Initialize ProxyAuth user for Nightingale
# This script ensures the admin user exists before Nightingale starts serving requests

set -e

echo "Waiting for PostgreSQL to be ready..."
until PGPASSWORD="${POSTGRES_PASSWORD:-your-postgres-password}" psql -h "${POSTGRES_HOST:-postgres}" -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nightingale}" -c '\q' 2>/dev/null; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done

echo "PostgreSQL is up - checking/creating ProxyAuth user"

# Use INSERT ... ON CONFLICT to handle duplicate user gracefully
PGPASSWORD="${POSTGRES_PASSWORD:-your-postgres-password}" psql -h "${POSTGRES_HOST:-postgres}" -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nightingale}" <<-EOSQL
  -- Insert admin user with ON CONFLICT clause to handle duplicates
  INSERT INTO users (username, nickname, password, phone, email, portrait, roles, contacts, maintainer, create_at, create_by, update_at, update_by, belong, last_active_time)
  VALUES ('admin', 'admin', '', '', '', '', 'Admin', NULL, 0, EXTRACT(EPOCH FROM NOW())::BIGINT, 'system', EXTRACT(EPOCH FROM NOW())::BIGINT, 'system', '', 0)
  ON CONFLICT (username) DO NOTHING;
EOSQL

echo "ProxyAuth user initialization complete"
