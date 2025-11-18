E2E validation: real-data dashboards (SaltStack & Slurm)

This short guide explains how to run the Playwright checks that assert dashboards use real data (no demo fallbacks) at your host, e.g. http://192.168.18.137:8080.

Prerequisites
- A normal user can log in. Defaults used by specs: user01 / user01-pass
- Salt: set a password so backend can authenticate via Salt NetAPI file eauth
  - In .env set SALT_API_PASSWORD to a strong password; SALT_API_USERNAME defaults to saltapi
  - Backend composes Salt API URL from SALT_API_SCHEME + SALT_MASTER_HOST + SALT_API_PORT when SALTSTACK_MASTER_URL is empty (default)
- Slurm: backend shells out to sinfo/squeue; ensure these tools exist inside backend container (or switch backend to a base image with slurm clients)

Run tests
- Install browsers once: npx playwright install --with-deps chromium
- Execute: BASE_URL=http://192.168.18.137:8080 npx playwright test test/e2e/specs --config=test/e2e/playwright.config.js
- Specs of interest: test/e2e/specs/saltstack-no-demo.spec.js and test/e2e/specs/slurm-no-demo.spec.js

Troubleshooting
- 502 on Salt: ensure SALT_API_PASSWORD is set and http://saltstack:8002/ is reachable from backend
- 502 on Slurm: verify sinfo/squeue presence inside backend container; otherwise backend returns 502 (no demo)
- Login failures: docker-compose.override.yml sets E2E_ALLOW_FAKE_LDAP=true in backend for dev; override E2E_USER/E2E_PASS in your shell if needed
