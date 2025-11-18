#!/bin/bash
# Initialize Nightingale datasource if not exists
# This script should be run after Nightingale database is ready

set -e

echo "Checking Nightingale datasource configuration..."

# Wait for database to be ready
until docker-compose exec -T postgres pg_isready -U postgres -d nightingale > /dev/null 2>&1; do
  echo "Waiting for database to be ready..."
  sleep 2
done

echo "Database is ready. Checking datasource..."

# Check if datasource exists
DATASOURCE_COUNT=$(docker-compose exec -T postgres psql -U postgres -d nightingale -t -c "SELECT COUNT(*) FROM datasource;" 2>/dev/null | tr -d ' ')

if [ "$DATASOURCE_COUNT" -eq "0" ]; then
  echo "No datasource found. Creating default Prometheus datasource..."
  
  docker-compose exec -T postgres psql -U postgres -d nightingale <<EOF
INSERT INTO datasource (
  id, 
  name, 
  description, 
  category, 
  plugin_id, 
  plugin_type, 
  plugin_type_name, 
  cluster_name, 
  settings, 
  status, 
  http, 
  auth, 
  is_default, 
  created_at, 
  created_by, 
  updated_at, 
  updated_by, 
  identifier
)
VALUES (
  1,
  'Default Prometheus',
  'Built-in Prometheus data source for AI-Infra-Matrix monitoring',
  'prometheus',
  0,
  'prometheus',
  'Prometheus',
  'Default',
  '{}',
  'enabled',
  '{"url": "http://nightingale:17000", "timeout": 30, "dial_timeout": 3, "max_idle_conns_per_host": 100, "tls": {"skip_tls_verify": false}}',
  '{"basic_auth": false, "basic_auth_user": "", "basic_auth_password": ""}',
  true,
  extract(epoch from now())::bigint,
  'system',
  extract(epoch from now())::bigint,
  'system',
  'default-prometheus'
)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  is_default = EXCLUDED.is_default,
  status = EXCLUDED.status,
  http = EXCLUDED.http,
  updated_at = extract(epoch from now())::bigint,
  updated_by = 'system';
EOF

  echo "✓ Default datasource created successfully"
else
  echo "✓ Datasource already exists (count: $DATASOURCE_COUNT)"
fi

# Verify datasource
echo ""
echo "Current datasources:"
docker-compose exec -T postgres psql -U postgres -d nightingale -c "SELECT id, name, plugin_type, status, is_default FROM datasource;"

echo ""
echo "✓ Nightingale datasource initialization complete"
