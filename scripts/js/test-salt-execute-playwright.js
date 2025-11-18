#!/usr/bin/env node

// Playwright E2E: SaltStack /saltstack 执行命令（自定义 Bash/Python）
// Env:
//   BASE_URL=http://192.168.0.200:8080
//   ADMIN_USER=admin ADMIN_PASS=admin123
//   HEADLESS=true|false
//   TARGET=*
//   LANGUAGE=bash|python
//   CODE=echo hello

const { chromium, request } = require('playwright');

function log(msg) { console.log(`[salt-exec-e2e] ${msg}`); }
function fail(msg, code = 1) { console.error(`[salt-exec-e2e] ERROR: ${msg}`); process.exit(code); }

(async () => {
  const BASE = process.env.BASE_URL || 'http://localhost:8080';
  const USER = process.env.ADMIN_USER || 'admin';
  const PASS = process.env.ADMIN_PASS || 'admin123';
  const HEADLESS = (process.env.HEADLESS || 'true').toLowerCase() !== 'false';
  const TARGET = process.env.TARGET || '*';
  const LANGUAGE = (process.env.LANGUAGE || 'bash').toLowerCase();
  const CODE = process.env.CODE || (LANGUAGE === 'bash' ? 'echo hello-from-bash' : 'print("hello-from-python")');

  log(`Base URL: ${BASE}`);

  const api = await request.newContext({ baseURL: `${BASE}/api/` });
  const resp = await api.post('auth/login', { data: { username: USER, password: PASS }, timeout: 20000 });
  if (!resp.ok()) fail(`Login failed ${resp.status()}: ${await resp.text()}`);
  const { token } = await resp.json();
  if (!token) fail('No token returned');
  log('Login succeeded');

  const browser = await chromium.launch({ headless: HEADLESS });
  const context = await browser.newContext();
  await context.addInitScript((tkn) => { try { localStorage.setItem('token', tkn); } catch(e){} }, token);
  const url = new URL(BASE);
  await context.addCookies([{ name: 'ai_infra_token', value: token, domain: url.hostname, path: '/', httpOnly: true }]);
  const page = await context.newPage();

  log('Open /saltstack');
  await page.goto(`${BASE}/saltstack`, { waitUntil: 'domcontentloaded' });

  log('Open 执行命令 modal');
  await page.getByRole('button', { name: '执行命令' }).click();
  await page.waitForSelector('text=执行自定义命令（Bash / Python）', { timeout: 10000 });

  // Fill target/language/code
  await page.locator('input[placeholder="例如: * 或 compute* 或 ai-infra-web-01"]').fill(TARGET);
  await page.getByRole('combobox').first().click();
  await page.locator('div[role="option"]').filter({ hasText: LANGUAGE === 'bash' ? 'Bash' : 'Python' }).click();
  await page.locator('textarea[placeholder="# 在此粘贴脚本..."]').fill(CODE);

  log('Submit execution');
  await page.getByRole('button', { name: '执行' }).click();

  // Wait for some events to show up
  const deadline = Date.now() + 60000;
  let gotOutput = false;
  while (Date.now() < deadline) {
    const text = await page.locator('div[style*="background: rgb(11, 16, 33)"]').innerText().catch(() => '');
    if (text && /命令输出|step/.test(text)) { gotOutput = true; break; }
    await page.waitForTimeout(1000);
  }
  if (!gotOutput) {
    const debug = await api.get('saltstack/_debug', { headers: { Authorization: `Bearer ${token}` }});
    log('Debug: ' + (await debug.text()).slice(0,1000));
    await browser.close();
    fail('No execution output observed in time');
  }

  log('✅ Custom command execution produced output');
  await browser.close();
  process.exit(0);
})().catch(err => fail(err?.stack || String(err)));
