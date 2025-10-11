#!/usr/bin/env node

// Playwright E2E test for SaltStack dashboard
// Usage:
//   BASE_URL=http://192.168.0.200:8080 ADMIN_USER=admin ADMIN_PASS=admin123 node test-saltstack-playwright.js
// Env:
//   BASE_URL         Base URL of the portal (default: http://localhost:8080)
//   ADMIN_USER       Login username (default: admin)
//   ADMIN_PASS       Login password (default: admin123)
//   HEADLESS         true|false (default: true)
//   TEST_TOKEN       If provided, skip login and use this JWT directly

const { chromium, request } = require('playwright');

function log(msg) { console.log(`[saltstack-e2e] ${msg}`); }
function fail(msg, code = 1) { console.error(`[saltstack-e2e] ERROR: ${msg}`); process.exit(code); }

(async () => {
  const BASE = process.env.BASE_URL || 'http://localhost:8080';
  const USER = process.env.ADMIN_USER || 'admin';
  const PASS = process.env.ADMIN_PASS || 'admin123';
  const HEADLESS = (process.env.HEADLESS || 'true').toLowerCase() !== 'false';
  let token = process.env.TEST_TOKEN || '';

  log(`Base URL: ${BASE}`);

  // Authenticate to get JWT token if not provided
  if (!token) {
    log('Attempting API login...');
  const api = await request.newContext({ baseURL: `${BASE}/api/` });
    const resp = await api.post('auth/login', {
      data: { username: USER, password: PASS },
      timeout: 15000,
    });
    if (!resp.ok()) {
      const body = await resp.text();
      fail(`Login failed (${resp.status()}): ${body}`);
    }
    const json = await resp.json();
    token = json.token || json?.data?.token;
    if (!token) fail('No token returned from login response');
    log('Login succeeded, token acquired');
  } else {
    log('Using token from TEST_TOKEN');
  }

  // Launch browser
  const browser = await chromium.launch({ headless: HEADLESS });
  const context = await browser.newContext();

  // Propagate token to frontend before any page scripts run
  await context.addInitScript((tkn) => {
    try {
      localStorage.setItem('token', tkn);
    } catch (e) {}
  }, token);
  // Also add a cookie for gateway-side SSO helpers (not required by axios, but helpful)
  try {
    const url = new URL(BASE);
    await context.addCookies([{
      name: 'ai_infra_token',
      value: token,
      domain: url.hostname,
      path: '/',
      httpOnly: true,
      secure: false,
      sameSite: 'Lax',
    }]);
  } catch (e) {
    // ignore cookie set failure
  }

  const page = await context.newPage();

  // Navigate to SaltStack dashboard and wait for API response
  log('Opening SaltStack dashboard...');
  const statusPromise = page.waitForResponse(r => r.url().includes('/api/saltstack/status'), { timeout: 20000 });
  await page.goto(`${BASE}/saltstack`, { waitUntil: 'domcontentloaded' });

  const statusResp = await statusPromise;
  if (!statusResp.ok()) {
    const body = await statusResp.text();
    await browser.close();
    fail(`/api/saltstack/status returned ${statusResp.status()}: ${body}`);
  }
  const statusJson = await statusResp.json();
  const statusData = statusJson?.data || statusJson;
  log(`Status payload: status=${statusData?.status}, demo=${statusData?.demo}`);

  if (statusData?.status === 'demo' || statusData?.demo === true) {
    await browser.close();
    fail('Dashboard is in demo mode, expected real data');
  }
  if (statusData?.status !== 'connected') {
    await browser.close();
    fail(`Expected status=connected, got ${statusData?.status}`);
  }

  // Sanity: check key UI bits present
  await page.waitForSelector('text=基本信息', { timeout: 10000 });
  await page.waitForSelector('text=服务状态', { timeout: 10000 });
  await page.waitForSelector('text=密钥统计', { timeout: 10000 });

  // Fetch minions and jobs endpoints to ensure they are healthy
  const [minionsResp, jobsResp] = await Promise.all([
    page.waitForResponse(r => r.url().includes('/api/saltstack/minions') && r.request().method() === 'GET', { timeout: 20000 }),
    page.waitForResponse(r => r.url().includes('/api/saltstack/jobs') && r.request().method() === 'GET', { timeout: 20000 }).catch(() => null),
  ]);

  if (!minionsResp?.ok()) {
    const body = minionsResp ? await minionsResp.text() : 'no response';
    await browser.close();
    fail(`/api/saltstack/minions failed: ${minionsResp?.status()} ${body}`);
  }
  const minionsJson = await minionsResp.json();
  const minions = minionsJson?.data || [];
  log(`Minions loaded: ${Array.isArray(minions) ? minions.length : 0}`);

  if (statusData?.connected_minions !== undefined && Array.isArray(minions)) {
    if (statusData.connected_minions < 0) {
      await browser.close();
      fail(`Invalid connected_minions: ${statusData.connected_minions}`);
    }
  }

  // Optional debug endpoint
  try {
    const dbg = await page.request.get(`${BASE}/api/saltstack/_debug`, {
      headers: { Authorization: `Bearer ${token}` },
      timeout: 15000,
    });
    if (dbg.ok()) {
      const dj = await dbg.json();
      const d = dj?.data || {};
      log(`Debug: tcp_ok=${d?.tcp_dial?.ok}, manage_up=${d?.manage_status?.up_count}, accepted=${(d?.key_list_all?.accepted||[]).length}`);
    } else {
      log(`Debug endpoint not available (${dbg.status()}) - continuing`);
    }
  } catch (e) {
    log(`Debug endpoint error: ${e.message}`);
  }

  log('✅ SaltStack dashboard E2E passed');
  await browser.close();
  process.exit(0);
})().catch(err => {
  fail(err?.stack || String(err));
});
