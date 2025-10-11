// @ts-nocheck
/* eslint-disable */
// SaltStack 执行按钮修复验证测试

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://192.168.0.200:8080';

// 辅助函数：登录
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

// 等待 SaltStack 页面加载
async function waitForSaltStackPageLoad(page) {
  try {
    await page.waitForSelector('text=加载SaltStack状态...', { state: 'hidden', timeout: 15000 });
  } catch (e) {
    // 可能加载很快，没有显示 loading
  }
  await expect(page.getByText('SaltStack 配置管理')).toBeVisible({ timeout: 15000 });
}

test('执行命令后按钮应该正确停止加载状态', async ({ page }) => {
  // 登录
  await loginIfNeeded(page);
  
  // 导航到 SaltStack 页面
  await page.goto(BASE + '/saltstack');
  await waitForSaltStackPageLoad(page);

  // 监听控制台日志
  const logs = [];
  page.on('console', msg => {
    const text = msg.text();
    if (text.includes('[SSE')) {
      logs.push(text);
      console.log('📝', text);
    }
  });

  // 打开执行命令对话框
  await page.getByRole('button', { name: /执行命令/ }).click();
  await expect(page.getByText('执行自定义命令')).toBeVisible();

  // 填写表单
  const codeTextarea = page.getByLabel('代码');
  await codeTextarea.clear();
  await codeTextarea.fill('hostname');
  
  // 确保目标节点为 *
  const targetInput = page.getByLabel('目标节点');
  await targetInput.clear();
  await targetInput.fill('*');
  
  // 点击执行按钮
  const executeButton = page.getByRole('button', { name: /执 行/ });
  await executeButton.click();

  console.log('⏳ 等待命令执行完成...');
  
  // 验证按钮变为禁用状态（loading）
  await expect(executeButton).toBeDisabled({ timeout: 2000 });
  console.log('✅ 按钮已进入 loading 状态');
  
  // 等待执行完成（按钮恢复可用）- 这是修复的关键测试点
  await expect(executeButton).toBeEnabled({ timeout: 35000 });
  console.log('✅ 按钮已恢复可用状态');
  
  // 验证看到完成消息
  const completedVisible = await page.locator('text=/执行完成|complete/').isVisible();
  expect(completedVisible).toBeTruthy();
  
  console.log('📋 收集到的 SSE 日志数量:', logs.length);
  logs.forEach(log => console.log('   ', log));
});

test('验证 SSE 事件流', async ({ page }) => {
  // 登录
  await loginIfNeeded(page);
  
  // 导航到 SaltStack 页面
  await page.goto(BASE + '/saltstack');
  await waitForSaltStackPageLoad(page);
  
  const events = [];
  
  // 监听控制台日志以捕获 SSE 事件
  page.on('console', msg => {
    const text = msg.text();
    if (text.includes('[SSE事件]')) {
      // 提取事件类型
      const match = text.match(/\[SSE事件\]\s+(\w+)/);
      if (match) {
        events.push(match[1]);
        console.log('📨 SSE 事件:', match[1]);
      }
    }
    if (text.includes('[SSE]')) {
      console.log('🔍 SSE 日志:', text);
    }
  });

  // 打开执行命令对话框
  await page.getByRole('button', { name: /执行命令/ }).click();
  await expect(page.getByText('执行自定义命令')).toBeVisible();
  
  // 填写表单
  const codeTextarea = page.getByLabel('代码');
  await codeTextarea.clear();
  await codeTextarea.fill('echo test');
  
  const targetInput = page.getByLabel('目标节点');
  await targetInput.clear();
  await targetInput.fill('*');
  
  // 执行命令
  const executeButton = page.getByRole('button', { name: /执 行/ });
  await executeButton.click();

  // 等待完成（按钮恢复可用）
  await expect(executeButton).toBeEnabled({ timeout: 35000 });

  console.log('📊 收到的所有事件:', events);
  
  // 验证事件流包含 complete
  expect(events.length).toBeGreaterThan(0);
  expect(events).toContain('complete');
  
  console.log('✅ SSE 事件流验证通过');
});
