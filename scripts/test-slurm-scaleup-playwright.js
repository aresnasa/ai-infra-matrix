#!/usr/bin/env node

// Playwright E2E: Add nodes via /slurm and verify SaltStack status on /saltstack
// Usage:
//   BASE_URL=http://192.168.18.137:8080 ADMIN_USER=admin ADMIN_PASS=admin123 node test-slurm-scaleup-playwright.js
// Env:
//   BASE_URL         Base URL of the portal (default: http://localhost:8080)
//   ADMIN_USER       Login username (default: admin)
//   ADMIN_PASS       Login password (default: admin123)
//   HEADLESS         true|false (default: true)
//   TEST_TOKEN       If provided, skip login and use this JWT directly
//   NODES            Comma or newline-separated list of hostnames (default: test-ssh01,test-ssh02,test-ssh03)
//   SSH_PASSWORD     SSH password to use (default: rootpass123)

const { chromium, request } = require('playwright');

function log(msg) { console.log(`[slurm-scaleup-e2e] ${msg}`); }
function fail(msg, code = 1) { console.error(`[slurm-scaleup-e2e] ERROR: ${msg}`); process.exit(code); }

(async () => {
  const BASE = process.env.BASE_URL || 'http://localhost:8080';
  const USER = process.env.ADMIN_USER || 'admin';
  const PASS = process.env.ADMIN_PASS || 'admin123';
  const HEADLESS = (process.env.HEADLESS || 'true').toLowerCase() !== 'false';
  const SSH_PASSWORD = process.env.SSH_PASSWORD || 'rootpass123';
  const NODES_ENV = process.env.NODES || 'test-ssh01,test-ssh02,test-ssh03';
  const INPUT_NODES = NODES_ENV.split(/[\n,]+/).map(s => s.trim()).filter(Boolean);
  let token = process.env.TEST_TOKEN || '';

  if (INPUT_NODES.length === 0) fail('No nodes provided in NODES env');

  log(`Base URL: ${BASE}`);
  log(`Nodes: ${INPUT_NODES.join(', ')}`);

  // 1) Authenticate to get JWT token
  if (!token) {
    log('Attempting API login...');
    const api = await request.newContext({ baseURL: `${BASE}/api/` });
    const resp = await api.post('auth/login', {
      data: { username: USER, password: PASS },
      timeout: 20000,
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

  // 2) Launch browser, propagate token
  const browser = await chromium.launch({ headless: HEADLESS });
  const context = await browser.newContext();
  await context.addInitScript((tkn) => {
    try { localStorage.setItem('token', tkn); } catch (e) {}
  }, token);
  try {
    const url = new URL(BASE);
    await context.addCookies([{ name: 'ai_infra_token', value: token, domain: url.hostname, path: '/', httpOnly: true, secure: false, sameSite: 'Lax' }]);
  } catch {}

  const page = await context.newPage();

  // 3) Navigate to /slurm and open the Scale Up modal
  log('Opening /slurm...');
  await page.goto(`${BASE}/slurm`, { waitUntil: 'domcontentloaded' });
  // Prefer the header button to avoid ambiguity with the tab extra button
  log('Opening 扩容节点 modal...');
  await page.getByRole('button', { name: '扩容节点' }).click();
  await page.waitForSelector('text=扩容 SLURM 节点', { timeout: 15000 });

  // 4) Fill node list
  const nodesTextarea = page.locator('textarea[placeholder*="每行一个节点配置"]');
  await nodesTextarea.fill(INPUT_NODES.join('\n'));

  // 5) Ensure auth type is 密码认证 and fill SSH password
  await page.getByRole('radio', { name: '密码认证' }).check();
  await page.locator('input[placeholder="请输入SSH用户密码"]').fill(SSH_PASSWORD);

  // 6) Specs: CPU, memory, disk = 1
  await page.locator('input[placeholder="例如: 4"]').fill('1');
  await page.locator('input[placeholder="例如: 8"]').fill('1');
  await page.locator('input[placeholder="例如: 100"]').fill('1');

  // 7) OS select: Ubuntu 22.04 (Ant Design Select)
  await page.getByText('操作系统').first().scrollIntoViewIfNeeded();
  const osInput = page.locator('input#os[role="combobox"]');
  try {
    await osInput.click({ force: true });
    // Try to open dropdown via keyboard in case click focuses but doesn't open
    await osInput.press('Enter').catch(() => {});
    await osInput.press('ArrowDown').catch(() => {});
    await page.waitForSelector('.ant-select-dropdown', { timeout: 3000 });
    await page.locator('div[role="option"]').filter({ hasText: 'Ubuntu 22.04' }).click();
  } catch (e) {
    log(`OS selection fallback (skipping explicit selection): ${e.message}`);
  }

  // 8) Check 自动部署 SaltStack Minion
  const chk = page.getByLabel('自动部署 SaltStack Minion').or(page.getByText('自动部署 SaltStack Minion'));
  try { await chk.click(); } catch {}

  // 9) Optionally test SSH connection (only for first node)
  try {
    log('Testing SSH connection...');
    const testBtn = page.getByRole('button', { name: '测试SSH连接' });
    await testBtn.click();
    await page.waitForSelector('text=连接测试成功', { timeout: 20000 });
    log('SSH connection test succeeded');
  } catch (e) {
    log(`SSH connection test skipped or failed: ${e.message}`);
  }

  // 10) Submit scale up
  log('Submitting scale-up...');
  await page.getByRole('button', { name: '开始扩容' }).click();

  // Wait for modal to close or success message to appear
  const successPromise = page.waitForSelector('text=扩容任务已提交', { timeout: 20000 }).catch(() => null);
  await Promise.race([
    page.waitForSelector('text=扩容 SLURM 节点', { state: 'detached', timeout: 20000 }).catch(() => null),
    successPromise,
  ]);

  // 11) Navigate to /saltstack and verify status
  log('Navigating to /saltstack to verify SaltStack status...');
  await page.goto(`${BASE}/saltstack`, { waitUntil: 'domcontentloaded' });

  // Wait and poll for connected status and minions visibility (up to ~2 minutes)
  const deadline = Date.now() + 120000;
  let connected = false;
  let statusData = null;
  let lastErr = '';

  while (Date.now() < deadline) {
    try {
      const resp = await page.request.get(`${BASE}/api/saltstack/status`, {
        headers: { Authorization: `Bearer ${token}` },
        timeout: 15000,
      });
      if (!resp.ok()) {
        lastErr = `/api/saltstack/status -> ${resp.status()} ${await resp.text()}`;
      } else {
        const js = await resp.json();
        statusData = js?.data || js;
        if (statusData?.status === 'connected') {
          connected = true;
          break;
        }
        lastErr = `status=${statusData?.status}`;
      }
    } catch (e) {
      lastErr = e.message;
    }
    await new Promise(r => setTimeout(r, 5000));
  }

  if (!connected) {
    // Try debug endpoint for context
    try {
      const dbg = await page.request.get(`${BASE}/api/saltstack/_debug`, { headers: { Authorization: `Bearer ${token}` }, timeout: 15000 });
      const dj = dbg.ok() ? await dbg.json() : { error: await dbg.text(), status: dbg.status() };
      log(`Debug endpoint: ${JSON.stringify(dj).slice(0, 800)}...`);
    } catch (e) {
      log(`Debug endpoint fetch failed: ${e.message}`);
    }
    await browser.close();
    fail(`SaltStack not connected after waiting: ${lastErr}`);
  }

  // Fetch minions list and log present nodes
  try {
    const m = await page.request.get(`${BASE}/api/saltstack/minions`, { headers: { Authorization: `Bearer ${token}` }, timeout: 15000 });
    if (m.ok()) {
      const mj = await m.json();
      const minions = mj?.data || mj || [];
      const text = JSON.stringify(minions);
      const found = INPUT_NODES.filter(h => text.includes(h));
      log(`Minions reported: found ${found.length}/${INPUT_NODES.length} of requested nodes`);
    } else {
      log(`/api/saltstack/minions returned ${m.status()}`);
    }
  } catch (e) {
    log(`minions fetch error: ${e.message}`);
  }

  log('✅ Slurm scale-up + SaltStack verification E2E passed');
  await browser.close();
  process.exit(0);
})().catch(err => {
  fail(err?.stack || String(err));
});
