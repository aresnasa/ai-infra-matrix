// @ts-nocheck
/* eslint-disable */
// SaltStack 执行按钮修复验证测试
// 测试修复后的执行按钮能否正确停止 loading 状态

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://localhost:8080';

// 辅助函数（供所有测试使用）
async function loginIfNeeded(page) {
  await page.goto(BASE + '/');
  
  const isLoggedIn = await page.locator('text=admin').isVisible().catch(() => false);
  
  if (!isLoggedIn) {
    const hasLoginTab = await page.getByRole('tab', { name: '登录' }).isVisible().catch(() => false);
    
    if (hasLoginTab) {
      const user = process.env.E2E_USER || 'admin';
      const pass = process.env.E2E_PASS || 'admin123';
      
      await page.getByPlaceholder('用户名').fill(user);
      await page.getByPlaceholder('密码').fill(pass);
      await page.getByRole('button', { name: /登\s*录/ }).click();
      
      await expect(page).toHaveURL(/\/(projects|dashboard|saltstack)?$/, { timeout: 10000 });
      await page.waitForLoadState('load');
    }
  }
}

// 等待 SaltStack 页面数据加载完成
async function waitForSaltStackPageLoad(page) {
  // 等待加载状态消失
  try {
    await page.waitForSelector('text=加载SaltStack状态...', { state: 'hidden', timeout: 15000 });
  } catch (e) {
    // Loading text might not appear if data loads quickly
  }
  
  // 等待页面标题出现
  await expect(page.getByText('SaltStack 配置管理')).toBeVisible({ timeout: 15000 });
}

test.describe('SaltStack 执行按钮修复验证', () => {
  test.beforeEach(async ({ page }) => {
    // 每个测试前先登录
    await loginIfNeeded(page);
  });

  test('【修复验证】执行按钮应该在命令完成后正确停止 loading', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 监听控制台日志以捕获 SSE 事件
    const sseEvents = [];
    page.on('console', msg => {
      const text = msg.text();
      if (text.includes('[SSE事件]')) {
        const match = text.match(/\[SSE事件\]\s+(\w+)/);
        if (match) {
          sseEvents.push(match[1]);
          console.log('📨 收到 SSE 事件:', match[1]);
        }
      }
      if (text.includes('[SSE]')) {
        console.log('🔍', text);
      }
    });
    
    // 打开执行命令对话框
    await page.getByRole('button', { name: /执行命令/ }).click();
    
    // 等待对话框加载
    await expect(page.getByText('执行自定义命令')).toBeVisible();
    
    // 清空默认代码并输入测试命令
    const codeTextarea = page.getByLabel('代码');
    await codeTextarea.clear();
    await codeTextarea.fill('hostname');
    
    // 确保目标节点为 * (所有节点)
    const targetInput = page.getByLabel('目标节点');
    await targetInput.clear();
    await targetInput.fill('*');
    
    // 点击执行按钮
    const executeButton = page.getByRole('button', { name: /执 行/ });
    await executeButton.click();
    
    console.log('⏳ 开始执行，等待按钮进入 loading 状态...');
    
    // 验证执行按钮显示 loading 状态
    await expect(executeButton).toBeDisabled({ timeout: 2000 });
    console.log('✅ 按钮已进入 loading 状态（禁用）');
    
    // 等待执行完成（按钮恢复可用）- 这是修复的关键测试点
    console.log('⏳ 等待命令执行完成，按钮应该恢复可用...');
    await expect(executeButton).toBeEnabled({ timeout: 35000 });
    console.log('✅ 按钮已恢复可用状态 - 修复验证成功！');
    
    // 验证看到完成消息
    const completedVisible = await page.locator('text=/执行完成|complete/').isVisible();
    expect(completedVisible).toBeTruthy();
    console.log('✅ 看到执行完成消息');
    
    // 验证 SSE 事件流
    console.log('📊 收到的 SSE 事件:', sseEvents);
    expect(sseEvents).toContain('complete');
    console.log('✅ SSE 事件流包含 complete 事件');
  });
});
