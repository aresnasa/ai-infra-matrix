#!/bin/bash

# SaltStack Integration Test Runner
# Runs Playwright tests for SaltStack minion installation and management

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🧪 SaltStack Integration Test Suite"
echo "===================================="
echo ""

# Load environment variables
if [ -f "$PROJECT_ROOT/.env" ]; then
    source "$PROJECT_ROOT/.env"
    echo "✓ Loaded environment from .env"
else
    echo "⚠️  Warning: .env file not found"
fi

# Check if AppHub is running
APPHUB_URL="${APPHUB_URL:-http://192.168.0.200:53434}"
echo ""
echo "🔍 Checking AppHub availability..."
if curl -s --head --fail "$APPHUB_URL" > /dev/null 2>&1; then
    echo "✓ AppHub is accessible at $APPHUB_URL"
else
    echo "❌ AppHub is not accessible at $APPHUB_URL"
    echo "   Please ensure AppHub container is running:"
    echo "   docker-compose up -d apphub"
    exit 1
fi

# Check if SaltStack packages are available
echo ""
echo "📦 Verifying SaltStack packages in AppHub..."
if curl -s --head --fail "$APPHUB_URL/pkgs/saltstack-deb/Packages.gz" > /dev/null 2>&1; then
    echo "✓ SaltStack deb package index found"
else
    echo "❌ SaltStack deb package index not found"
    echo "   Please rebuild AppHub with SaltStack packages:"
    echo "   ./build.sh build apphub --no-cache"
    exit 1
fi

# Verify individual packages
echo ""
echo "🔍 Checking individual packages..."
PACKAGES=(
    "salt-common_3007.8_arm64.deb"
    "salt-minion_3007.8_arm64.deb"
    "salt-master_3007.8_arm64.deb"
)

for pkg in "${PACKAGES[@]}"; do
    if curl -s --head --fail "$APPHUB_URL/pkgs/saltstack-deb/$pkg" > /dev/null 2>&1; then
        echo "  ✓ $pkg"
    else
        echo "  ❌ $pkg not found"
    fi
done

# Check Playwright installation
echo ""
echo "🎭 Checking Playwright installation..."
if ! command -v npx &> /dev/null; then
    echo "❌ npx not found. Please install Node.js and npm"
    exit 1
fi

# Install Playwright browsers if needed
echo ""
echo "🌐 Ensuring Playwright browsers are installed..."
npx --yes playwright install chromium

# Run the tests
echo ""
echo "🚀 Running SaltStack integration tests..."
echo "=========================================="
echo ""

cd "$PROJECT_ROOT"

# Run Playwright tests with custom config
npx --yes playwright test \
    test/e2e/specs/saltstack-integration.spec.js \
    --config=test/e2e/playwright.config.js \
    --reporter=html \
    --reporter=list

TEST_EXIT_CODE=$?

# Display results
echo ""
echo "=========================================="
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ All tests passed!"
else
    echo "❌ Some tests failed (exit code: $TEST_EXIT_CODE)"
fi

echo ""
echo "📊 Test results:"
echo "   - HTML report: playwright-report/index.html"
echo "   - Screenshots: test-screenshots/"
echo ""

# Open HTML report if available
if [ -f "playwright-report/index.html" ]; then
    echo "To view the HTML report, run:"
    echo "   npx playwright show-report"
fi

exit $TEST_EXIT_CODE
