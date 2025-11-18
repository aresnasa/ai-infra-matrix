// @ts-nocheck
/* eslint-disable */
// Verifies SaltStack dashboard is using real data (no demo banner)
const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://localhost:8080';

async function loginIfNeeded(page) {
  await page.goto(BASE + '/');
  if (await page.getByRole('tab', { name: '登录' }).isVisible().catch(() => false)) {
    const user = process.env.E2E_USER || 'user01';
    const pass = process.env.E2E_PASS || 'user01-pass';
    await page.getByPlaceholder('用户名').fill(user);
    await page.getByPlaceholder('密码').fill(pass);
    await page.getByRole('button', { name: '登录' }).click();
    await expect(page).toHaveURL(/\/(projects|)$/);
  }
}

test('SaltStack dashboard has no 演示 banner and loads real data', async ({ page }) => {
  await loginIfNeeded(page);
  await page.goto(BASE + '/saltstack');
  // Ensure page renders
  await expect(page.getByText('SaltStack 配置管理')).toBeVisible();
  // Demo banner should be absent
  await expect(page.getByText(/演示模式|显示SaltStack演示数据/)).toHaveCount(0);
  // Key stats should render (may be 0 but present)
  await expect(page.getByText('Master状态')).toBeVisible();
  await expect(page.getByText('在线Minions')).toBeVisible();
  await expect(page.getByText('API状态')).toBeVisible();
});
