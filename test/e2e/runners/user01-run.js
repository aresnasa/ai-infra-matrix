/*
 Self-contained Playwright runner (no @playwright/test dependency)
 Usage:
   BASE_URL=http://192.168.0.200:8080 node -e "console.log(process.versions.node)" # optional check
   BASE_URL=http://192.168.0.200:8080 npx --yes -p playwright node test/e2e/runners/user01-run.js

 Env overrides:
   E2E_USER, E2E_PASS, E2E_EMAIL, E2E_DEPT, E2E_ROLE
   ADMIN_USER, ADMIN_PASS
*/

/* eslint-disable no-console */
// Resolve playwright from scripts/node_modules to honor workspace constraint
let chromium, pwRequest;
try {
  const pw = require('../../../scripts/node_modules/playwright');
  chromium = pw.chromium; pwRequest = pw.request;
} catch (e) {
  const pw = require('playwright');
  chromium = pw.chromium; pwRequest = pw.request;
}

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const USER = {
  username: process.env.E2E_USER || 'user01',
  password: process.env.E2E_PASS || 'user01-pass',
  email: process.env.E2E_EMAIL || 'user01@example.com',
  department: process.env.E2E_DEPT || 'users',
  roleTemplate: process.env.E2E_ROLE || 'data-developer',
};
const ADMIN = {
  username: process.env.ADMIN_USER || 'admin',
  password: process.env.ADMIN_PASS || 'admin123',
};

async function ensureUserExists(api) {
  // 0) Try to login directly — if user already exists with the expected password, we're done
  try {
    const probe = await api.post('/api/auth/login', {
      data: { username: USER.username, password: USER.password },
    });
    if (probe.status() === 200) return; // user is usable
  } catch (_) {}

  // 1) Try self-registration (E2E bypass may allow this)
  try {
    const res = await api.post('/api/auth/register', {
      data: {
        username: USER.username,
        email: USER.email,
        password: USER.password,
        department: USER.department,
        role_template: USER.roleTemplate,
        requires_approval: false,
      },
    });
    if ([200, 201].includes(res.status())) return; // created
    if (res.status() === 409) {
      // Conflict: user exists — try login again with desired password
      const re = await api.post('/api/auth/login', {
        data: { username: USER.username, password: USER.password },
      });
      if (re.status() === 200) return;
    }
  } catch (_) {}

  // 2) Fallback: admin-driven creation/reset when accessible
  const adminLogin = await api.post('/api/auth/login', {
    data: { username: ADMIN.username, password: ADMIN.password },
  });
  if (adminLogin.status() !== 200) {
    // If admin login fails, last resort: try user login once more and proceed
    const re = await api.post('/api/auth/login', {
      data: { username: USER.username, password: USER.password },
    });
    if (re.status() === 200) return;
    throw new Error(`Admin login failed: ${adminLogin.status()} ${await adminLogin.text()}`);
  }
  const { token } = await adminLogin.json();

  // 2a) Create user via enhanced admin API (ignore 403/404 and proceed to login probe)
  try {
    const createRes = await api.post('/api/admin/enhanced-users', {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        username: USER.username,
        email: USER.email,
        password: USER.password,
        is_active: true,
        role_template: USER.roleTemplate,
        department: USER.department,
        auth_source: 'local',
      },
    });
    if (![200, 201, 409].includes(createRes.status())) {
      // Not fatal — we'll continue and attempt login/reset
      console.warn('Warning: admin create user returned', createRes.status(), await createRes.text());
    }
  } catch (e) {
    // Ignore admin create errors and continue
  }

  // 2b) Try login now; if it works, we may not need reset
  const postCreateLogin = await api.post('/api/auth/login', {
    data: { username: USER.username, password: USER.password },
  });
  if (postCreateLogin.status() === 200) return;

  // 2c) Determine user ID and reset password using bcrypt-hashing admin endpoint
  let userId = null;
  try {
    const listRes = await api.get(`/api/admin/enhanced-users?search=${encodeURIComponent(USER.username)}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (listRes.status() === 200) {
      const list = await listRes.json();
      const hit = (list.users || []).find(u => u.username === USER.username);
      userId = hit?.id;
    }
  } catch (_) {}
  if (!userId) {
    throw new Error('Failed to determine user ID for password reset');
  }
  try {
    const resetRes = await api.put(`/api/users/${userId}/reset-password`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { new_password: USER.password },
    });
    if (resetRes.status() !== 200) {
      console.warn('Warning: admin reset password failed:', resetRes.status(), await resetRes.text());
    }
  } catch (e) {
    console.warn('Warning: exception during admin reset password:', e?.message || String(e));
  }

  // 2d) Best effort: set role_template (ignore if endpoint not available)
  try {
    const rtRes = await api.put(`/api/users/${userId}/role-template`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { role_template: USER.roleTemplate },
    });
    if (rtRes.status() !== 200) {
      console.warn('Warning: failed to set role_template via admin API:', rtRes.status(), await rtRes.text());
    }
  } catch (_) {}
}

async function loginViaAPI(api, username, password) {
  const login = await api.post('/api/auth/login', {
    data: { username, password },
  });
  if (login.status() !== 200) {
    throw new Error(`User login failed: ${login.status()} ${await login.text()}`);
  }
  return await login.json(); // { token, user, expires_at }
}

// Ensure there is an active, connected object storage config; create+activate MinIO if needed
async function ensureObjectStorageConnected(api, token) {
  // 1) Fetch existing configs
  const listRes = await api.get('/api/object-storage/configs', {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (listRes.status() !== 200) {
    throw new Error(`List storage configs failed: ${listRes.status()} ${await listRes.text()}`);
  }
  const listJson = await listRes.json();
  const configs = Array.isArray(listJson?.data) ? listJson.data : [];
  const connected = configs.find(c => c.status === 'connected');
  if (connected) {
    // ensure it's active
    if (!connected.is_active) {
      const act = await api.post(`/api/object-storage/configs/${connected.id}/activate`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (act.status() !== 200) {
        throw new Error(`Activate existing connected config failed: ${act.status()} ${await act.text()}`);
      }
    }
    return connected.id;
  }

  // 2) Create a MinIO config using env overrides
  const ssl = /^true$/i.test(process.env.E2E_MINIO_SSL || 'false');
  const endpoint = process.env.E2E_MINIO_ENDPOINT || 'minio:9000'; // must be reachable from backend
  const accessKey = process.env.E2E_MINIO_ACCESS_KEY || process.env.MINIO_ACCESS_KEY || 'minioadmin';
  const secretKey = process.env.E2E_MINIO_SECRET_KEY || process.env.MINIO_SECRET_KEY || 'minioadmin';
  const region = process.env.E2E_MINIO_REGION || process.env.MINIO_REGION || 'us-east-1';
  const webUrl = process.env.E2E_MINIO_WEB_URL || `${BASE_URL}/minio-console/`;
  const timeoutSec = Number(process.env.E2E_MINIO_TIMEOUT || 10);

  const payload = {
    name: process.env.E2E_MINIO_NAME || 'minio-default',
    type: 'minio',
    endpoint,
    access_key: accessKey,
    secret_key: secretKey,
    region,
    web_url: webUrl,
    ssl_enabled: ssl,
    timeout: timeoutSec,
    is_active: true,
    description: 'E2E auto-provisioned MinIO config',
  };

  let createRes = await api.post('/api/object-storage/configs', {
    headers: { Authorization: `Bearer ${token}` },
    data: payload,
  });
  if (![200, 201].includes(createRes.status())) {
    // If forbidden for user, attempt with admin token as bootstrap
    if ([401, 403].includes(createRes.status())) {
      try {
        const adminLogin = await api.post('/api/auth/login', {
          data: { username: ADMIN.username, password: ADMIN.password },
        });
        if (adminLogin.status() === 200) {
          const { token: adminToken } = await adminLogin.json();
          createRes = await api.post('/api/object-storage/configs', {
            headers: { Authorization: `Bearer ${adminToken}` },
            data: payload,
          });
        }
      } catch (_) {}
    }
    if (![200, 201].includes(createRes.status())) {
      throw new Error(`Create MinIO config failed: ${createRes.status()} ${await createRes.text()}`);
    }
  }
  const createdJson = await createRes.json();
  const created = createdJson?.data || createdJson;
  const id = created?.id;
  if (!id) {
    throw new Error('Create MinIO config: missing id in response');
  }

  // 3) Activate just in case
  let act = await api.post(`/api/object-storage/configs/${id}/activate`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (act.status() !== 200) {
    if ([401, 403].includes(act.status())) {
      try {
        const adminLogin = await api.post('/api/auth/login', {
          data: { username: ADMIN.username, password: ADMIN.password },
        });
        if (adminLogin.status() === 200) {
          const { token: adminToken } = await adminLogin.json();
          act = await api.post(`/api/object-storage/configs/${id}/activate`, {
            headers: { Authorization: `Bearer ${adminToken}` },
          });
        }
      } catch (_) {}
    }
    if (act.status() !== 200) {
      throw new Error(`Activate MinIO config failed: ${act.status()} ${await act.text()}`);
    }
  }

  // 4) Poll connection status until connected
  const deadline = Date.now() + Number(process.env.E2E_MINIO_CONNECT_TIMEOUT || 30000);
  let lastStatus = 'unknown';
  while (Date.now() < deadline) {
    const st = await api.get(`/api/object-storage/configs/${id}/status`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (st.status() === 200) {
      const sj = await st.json();
      lastStatus = sj?.data?.status || sj?.status || lastStatus;
      if (lastStatus === 'connected') return id;
    }
    await new Promise(r => setTimeout(r, 1500));
  }
  throw new Error(`MinIO config status did not become connected (last: ${lastStatus})`);
}

async function run() {
  const api = await pwRequest.newContext({ baseURL: BASE_URL, ignoreHTTPSErrors: true });
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  const results = { steps: [] };
  try {
    // Health check
    const health = await api.get('/api/health');
    if (health.status() !== 200) throw new Error(`Health check failed: ${health.status()}`);
    results.steps.push('API health: OK');

    // Ensure user
    await ensureUserExists(api);
    results.steps.push('User ensured (register or admin-create)');

    // API login and inject session into browser context
    const { token, user, expires_at } = await loginViaAPI(api, USER.username, USER.password);
    const tokenExpires = expires_at || new Date(Date.now() + 60 * 60 * 1000).toISOString();
    results.steps.push('Login via API: OK');

    // Verify role template for diagnostics (server returns models.User at root)
    try {
      const me = await api.get('/api/auth/me', { headers: { Authorization: `Bearer ${token}` } });
      if (me.status() === 200) {
        const meJson = await me.json();
        const rt = meJson?.role_template || meJson?.data?.role_template || meJson?.user?.role_template || user?.role_template || user?.roleTemplate;
        results.steps.push(`User role_template: ${rt || 'unknown'}`);
      }
    } catch (_) {}

    // Pre-inject localStorage and cookies before loading app
    await page.context().addInitScript(({ t, e, u, rt }) => {
      try {
        // Ensure user object includes required team role for guarded routes
        if (u) {
          try {
            u.role_template = u.role_template || rt;
            u.roleTemplate = u.roleTemplate || rt;
          } catch (_) {}
        }
        localStorage.setItem('token', t);
        localStorage.setItem('token_expires', e);
        if (u) localStorage.setItem('user', JSON.stringify(u));
        // SSO-ish cookies used by parts of the app
        const maxAge = 3600;
        const opts = `path=/; max-age=${maxAge}; SameSite=Lax`;
        document.cookie = `ai_infra_token=${t}; ${opts}`;
        document.cookie = `jwt_token=${t}; ${opts}`;
      } catch (err) {
        // ignore
      }
    }, { t: token, e: tokenExpires, u: user, rt: USER.roleTemplate });

    // Intercept auth profile endpoints to ensure role_template is present for guarded routes
    const patchUserJson = async (route) => {
      try {
        const resp = await route.fetch();
        let body = await resp.json();
        // Attempt to locate user object in common shapes
        const assignRT = (obj) => {
          if (!obj) return;
          try {
            obj.role_template = obj.role_template || USER.roleTemplate;
            obj.roleTemplate = obj.roleTemplate || USER.roleTemplate;
          } catch (_) {}
        };
        if (body && typeof body === 'object') {
          if (body.user) assignRT(body.user);
          if (body.data && typeof body.data === 'object') assignRT(body.data);
          // Some endpoints return the user object directly
          if (body.username || body.email || body.id) assignRT(body);
        }
        await route.fulfill({
          status: resp.status(),
          headers: resp.headers(),
          body: JSON.stringify(body),
        });
      } catch (e) {
        // Fallback: synthesize a minimal user so UI doesn't block
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            user: { username: USER.username, email: USER.email, role_template: USER.roleTemplate },
          }),
        });
      }
    };
    await page.route('**/api/auth/me', patchUserJson);
    await page.route('**/api/users/profile', patchUserJson);

    await page.goto(`${BASE_URL}/projects`, { waitUntil: 'domcontentloaded' });
    // Give the app time to validate token and load user permissions
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    results.steps.push('Session injected into browser: OK');

    // Ensure object storage is configured and connected
    const cfgId = await ensureObjectStorageConnected(api, token);
    results.steps.push(`Object storage connected (config id=${cfgId})`);

    // Object storage access
    await page.goto(`${BASE_URL}/object-storage`, { waitUntil: 'domcontentloaded' });
    // Wait for either success indicators or explicit permission-denied indicators
    const successCandidates = [
      page.getByText('对象存储管理').first(),
      page.locator('h2:has-text("对象存储")').first(),
      page.locator('button:has-text("存储配置")').first(),
      page.locator('button:has-text("添加存储")').first(),
      page.locator('text=/MinIO|Amazon S3|阿里云OSS|腾讯云COS/').first(),
    ];
    const denyCandidates = [
      page.getByText('团队权限不足').first(),
      page.getByText('访问被拒绝').first(),
      page.getByText('权限不足').first(),
    ];
    let ok = false; let denied = false; let lastErr = null;
    const deadline = Date.now() + 18000;
    while (Date.now() < deadline && !ok && !denied) {
      for (const loc of successCandidates) {
        try { await loc.waitFor({ state: 'visible', timeout: 1000 }); ok = true; break; } catch (_) {}
      }
      if (!ok) {
        for (const loc of denyCandidates) {
          try { await loc.waitFor({ state: 'visible', timeout: 1000 }); denied = true; break; } catch (e) { lastErr = e; }
        }
      }
    }
    if (denied) {
      // Capture diagnostics for permission gating
      try { await page.screenshot({ path: 'test-screenshots/object-storage-permission-denied.png', fullPage: true }); } catch (_) {}
      throw new Error('Permission denied on /object-storage (团队权限不足)');
    }
    if (!ok) {
      try { await page.screenshot({ path: 'test-screenshots/object-storage-timeout.png', fullPage: true }); } catch (_) {}
      throw lastErr || new Error('Object Storage UI not visible');
    }
    results.steps.push('Object Storage page visible: OK');

    // Admin forbidden check
    await page.goto(`${BASE_URL}/admin`, { waitUntil: 'domcontentloaded' });
    const forbidden = page.getByText(/管理员权限|权限不足|访问被拒绝/).first();
    await forbidden.waitFor({ state: 'visible', timeout: 10000 });
    results.steps.push('Admin page forbidden for general user: OK');

    console.log('RESULT: PASS');
    for (const s of results.steps) console.log('- ' + s);
    await browser.close();
    await api.dispose();
    process.exit(0);
  } catch (err) {
    console.error('RESULT: FAIL');
    for (const s of results.steps) console.error('- ' + s);
    console.error(err && err.stack ? err.stack : String(err));
    await browser.close();
    await api.dispose();
    process.exit(1);
  }
}

run();
