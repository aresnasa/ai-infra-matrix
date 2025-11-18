// @ts-check
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
  // Some deployments might enforce LDAP and return 500/401; surface body for debugging
  const text = await res.text();
  throw new Error(`Failed to ensure user exists: ${res.status()} ${text}`);
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
