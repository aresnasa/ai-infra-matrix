#!/bin/bash
# Quick deployment script for Nightingale datasource fix
# Run this after initial deployment to ensure datasource is configured

set -e

echo "================================"
echo "Nightingale Datasource Fix"
echo "================================"
echo ""

# Check if docker-compose is running
if ! docker-compose ps | grep -q "Up"; then
  echo "⚠️  Warning: Some services may not be running"
  echo "Starting services first..."
  docker-compose up -d
  echo "Waiting for services to be ready..."
  sleep 10
fi

# Run the datasource initialization script
echo "Running datasource initialization..."
./scripts/init-nightingale-datasource.sh

echo ""
echo "================================"
echo "✓ Fix deployment complete!"
echo "================================"
echo ""
echo "You can now access Nightingale at:"
echo "  http://192.168.0.200:8080/monitoring"
echo ""
echo "To verify the fix, run:"
echo "  BASE_URL=http://192.168.0.200:8080 npx playwright test test/e2e/specs/verify-datasource-fix.spec.js --config=test/e2e/playwright.config.js"
