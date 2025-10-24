#!/bin/bash

# Monitoring Page 404 Fix - Quick Test Script
# This script runs all monitoring-related E2E tests

set -e

echo "======================================"
echo "Monitoring Page E2E Test Suite"
echo "======================================"
echo ""

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Playwright is available
if ! command -v npx &> /dev/null; then
    echo "❌ npx command not found. Please install Node.js and npm."
    exit 1
fi

# Test 1: 404 Debug Test
echo "${YELLOW}[Test 1/2] Running 404 Debug Test...${NC}"
echo "--------------------------------------"
npx playwright test test/e2e/specs/monitoring-404-debug.spec.js --config=test/e2e/playwright.config.js

echo ""
echo "${GREEN}✅ Test 1 completed${NC}"
echo ""

# Test 2: Complete Functionality Test
echo "${YELLOW}[Test 2/2] Running Complete Functionality Test...${NC}"
echo "--------------------------------------"
npx playwright test test/e2e/specs/monitoring-complete-test.spec.js --config=test/e2e/playwright.config.js

echo ""
echo "${GREEN}✅ Test 2 completed${NC}"
echo ""

# Summary
echo "======================================"
echo "${GREEN}All monitoring tests passed!${NC}"
echo "======================================"
echo ""
echo "Test Results:"
echo "  ✅ No 404 errors detected"
echo "  ✅ Monitoring iframe loads correctly"
echo "  ✅ All static assets (font, js, images) load successfully"
echo "  ✅ No JavaScript errors"
echo "  ✅ SSO integration working"
echo ""
echo "Screenshots saved to: test-screenshots/"
echo ""
