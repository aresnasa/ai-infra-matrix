// @ts-nocheck
/* eslint-disable */
// Note: This suite is executed with external Playwright (npx), not within repo deps.
const { test, expect } = require('@playwright/test');

const USER = {
  username: process.env.E2E_USER || 'user01',
  password: process.env.E2E_PASS || 'user01-pass',
  email: process.env.E2E_EMAIL || 'user01@example.com',
  department: process.env.E2E_DEPT || 'users',
  roleTemplate: process.env.E2E_ROLE || 'data-developer',
};

// Helper: perform login
async function login(page) {
  await page.goto('/');
  // AuthPage shows when not logged in
  await expect(page.getByRole('tab', { name: '登录' })).toBeVisible();
  await page.getByPlaceholder('用户名').fill(USER.username);
  await page.getByPlaceholder('密码').fill(USER.password);
  await page.getByRole('button', { name: '登录' }).click();
  // After login, the app typically routes to /projects
  await expect(page).toHaveURL(/\/projects|\/?$/);
}

// Helper: register new user via API to avoid UI fragility
async function registerViaAPI(request) {
  const base = process.env.BASE_URL || 'http://localhost:8080';
  // e2e bypass requires server env E2E_ALLOW_FAKE_LDAP=true
  const res = await request.post(`${base}/api/auth/register`, {
    data: {
      username: USER.username,
      email: USER.email,
      password: USER.password,
      department: USER.department,
      role_template: USER.roleTemplate,
      requires_approval: false,
    },
  });
  return res;
}

// If user exists, ignore conflict; otherwise register
async function ensureUserExists(request) {
  const res = await registerViaAPI(request);
  if (res.status() === 201) return;
  if (res.status() === 409) return; // already exists

  // If registration fails (likely due to LDAP requirement), try admin-driven creation
  const base = process.env.BASE_URL || 'http://localhost:8080';
  const adminUser = process.env.ADMIN_USER || 'admin';
  const adminPass = process.env.ADMIN_PASS || 'admin123';

  // Login as admin to get token
  const loginRes = await request.post(`${base}/api/auth/login`, {
    data: { username: adminUser, password: adminPass },
  });
  if (loginRes.status() !== 200) {
    const t = await loginRes.text();
    throw new Error(`Admin login failed: ${loginRes.status()} ${t}`);
  }
  const loginJson = await loginRes.json();
  const token = loginJson.token;

  // Create user via admin API
  const createRes = await request.post(`${base}/api/admin/enhanced-users`, {
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
    const t = await createRes.text();
    throw new Error(`Admin create user failed: ${createRes.status()} ${t}`);
  }
}

test.describe('General user flow: user01', () => {
  test('register (if needed) and login, then basic navigation', async ({ page, request }) => {
    await ensureUserExists(request);

    // Login through UI
    await login(page);

    // Verify allowed routes for data-developer
    await page.goto('/projects');
    await expect(page).toHaveURL(/\/projects/);

    // Access Object Storage overview
    await page.goto('/object-storage');
    await expect(page.getByText('对象存储管理')).toBeVisible();

    // Verify admin route is forbidden (AdminProtectedRoute shows 403 Result)
    await page.goto('/admin');
    await expect(page.getByText(/管理员权限|权限不足|访问被拒绝/)).toBeVisible();
  });
});
