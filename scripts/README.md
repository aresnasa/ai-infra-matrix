# Playwright E2E tests for Object Storage

This package runs a lightweight Playwright smoke suite to validate the Object Storage pages and the MinIO console proxy/iframe.

## Install

- Install dependencies in this folder:

  npm install

- Install Playwright browsers once (or when upgrading Playwright):

  npm run install-browsers

## Run

- Against local dev (localhost:8080):

  npm run test:local

- Against the deployed host (replace with yours if different):

  npm run test:remote

- Headed mode for debugging:

  npm run test:headed

- All browsers:

  npm run test:multi

## Env vars

- FRONTEND_URL: Base URL of the frontend (default <http://localhost:8080>)
- HEADLESS: true/false (default true)
- BROWSERS: comma-separated list (chromium,firefox,webkit)
- TIMEOUT: per-step timeout in ms (default 30000)
- SCREENSHOT: true/false to save screenshots (default true)

## Notes

- Tests attempt to log in via /api/auth/login with admin/admin123 by default. Override via TEST_USERNAME/TEST_PASSWORD if needed.
- Screenshots are saved to ./test-screenshots.
