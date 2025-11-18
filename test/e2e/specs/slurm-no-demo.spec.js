// @ts-nocheck
/* eslint-disable */
// Verifies Slurm dashboard is using real data (no demo banner)
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

test('Slurm dashboard has no 演示 banner and renders tables', async ({ page }) => {
  await loginIfNeeded(page);
  await page.goto(BASE + '/slurm');
  await expect(page.getByText('Slurm 集群管理')).toBeVisible();
  // Demo info alert should not exist
  await expect(page.getByText(/使用演示数据/)).toHaveCount(0);
  // Tables visible (even if empty)
  await expect(page.getByText('节点列表')).toBeVisible();
  await expect(page.getByText('作业队列')).toBeVisible();
});
