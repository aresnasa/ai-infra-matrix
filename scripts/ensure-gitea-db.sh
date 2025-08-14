#!/usr/bin/env bash
set -euo pipefail

# Ensure Gitea role and database exist in the running Postgres container.
# Useful when the Postgres data volume pre-dates the Gitea auto-init change
# (the init script only runs on first DB initialization).

GITEA_DB_NAME=${GITEA_DB_NAME:-gitea}
GITEA_DB_USER=${GITEA_DB_USER:-gitea}
GITEA_DB_PASSWD=${GITEA_DB_PASSWD:-gitea-password}
POSTGRES_SERVICE=${POSTGRES_SERVICE:-postgres}

psql_exec() {
  docker compose exec -T "${POSTGRES_SERVICE}" psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "$1"
}

# Create role if missing
# Use DO $$ ... $$ but escape the $$ so the shell doesn't substitute it
psql_exec "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${GITEA_DB_USER}') THEN CREATE USER ${GITEA_DB_USER} WITH LOGIN PASSWORD '${GITEA_DB_PASSWD}'; END IF; END \$\$;"

# Create DB if missing and grant privileges
docker compose exec -T "${POSTGRES_SERVICE}" psql -U postgres -d postgres <<SQL
SELECT 'CREATE DATABASE ${GITEA_DB_NAME} OWNER ${GITEA_DB_USER}' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${GITEA_DB_NAME}')\gexec
GRANT ALL PRIVILEGES ON DATABASE ${GITEA_DB_NAME} TO ${GITEA_DB_USER};
SQL

echo "Gitea database ensured: user='${GITEA_DB_USER}' db='${GITEA_DB_NAME}'"
