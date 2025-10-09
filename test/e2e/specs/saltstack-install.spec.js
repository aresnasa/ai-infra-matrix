// @ts-nocheck
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const API_BASE_URL = process.env.API_BASE_URL || `${BASE_URL.replace(/\/$/, '')}/api`;
const APPHUB_URL = process.env.APPHUB_URL || 'http://localhost:8090';
const ADMIN_USER = process.env.E2E_USER || 'admin';
const ADMIN_PASS = process.env.E2E_PASS || 'admin123';

const TEST_NODES = ['test-ssh01', 'test-ssh02', 'test-ssh03'];

async function loginAsAdmin(request) {
  const response = await request.post(`${API_BASE_URL}/auth/login`, {
    data: {
      username: ADMIN_USER,
      password: ADMIN_PASS,
    },
  });

  expect(response.ok()).toBeTruthy();
  const body = await response.json();
  expect(body).toHaveProperty('token');
}

function assertStepSuccess(results, stepName) {
  const match = results.find((result) => result.Name === stepName);
  expect(match, `step "${stepName}" should exist`).toBeTruthy();
  expect(match.Success, `step "${stepName}" should succeed`).toBeTruthy();
}

test.describe('SaltStack client installation', () => {
  test.describe.configure({ mode: 'serial' });
  test.beforeAll(async ({ request }) => {
    await loginAsAdmin(request);
  });

  test('installs SaltStack minion on test nodes via install-test-nodes API', async ({ request }) => {
    test.setTimeout(5 * 60 * 1000);

    const payload = {
      nodes: TEST_NODES,
      appHubURL: APPHUB_URL,
      enableSaltMinion: true,
      enableSlurmClient: false,
    };

    const response = await request.post(`${API_BASE_URL}/slurm/install-test-nodes`, {
      data: payload,
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.success).toBeTruthy();
    expect(Array.isArray(body.results)).toBeTruthy();

    for (const hostResult of body.results) {
      expect(hostResult.success).toBeTruthy();
      expect(Array.isArray(hostResult.steps)).toBeTruthy();
      assertStepSuccess(hostResult.steps, 'install_saltstack_minion');
      assertStepSuccess(hostResult.steps, 'configure_apt_source');
      assertStepSuccess(hostResult.steps, 'final_verification');
    }
  });

  test('SaltStack dashboard is accessible after installation', async ({ page }) => {
    test.setTimeout(2 * 60 * 1000);

    await page.goto(BASE_URL);

    if (await page.getByRole('tab', { name: '登录' }).isVisible().catch(() => false)) {
      await page.getByPlaceholder('用户名').fill(ADMIN_USER);
      await page.getByPlaceholder('密码').fill(ADMIN_PASS);
      await page.getByRole('button', { name: '登录' }).click();
    }

    await page.goto(`${BASE_URL.replace(/\/$/, '')}/saltstack`);
    await expect(page.getByText('SaltStack 配置管理')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText(/演示模式|显示SaltStack演示数据/)).toHaveCount(0);
  });
});
