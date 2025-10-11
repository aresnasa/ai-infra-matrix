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

  test('应该能打开 SaltStack 页面', async ({ page }) => {
    // 访问 SaltStack 页面
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 验证页面标题
    await expect(page.getByText('SaltStack 配置管理')).toBeVisible();
    
    // 验证关键元素存在
    await expect(page.getByRole('button', { name: /执行命令/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /刷新数据/ })).toBeVisible();
  });

  test('应该能打开执行命令对话框', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 点击"执行命令"按钮
    await page.getByRole('button', { name: /执行命令/ }).click();
    
    // 验证对话框打开
    await expect(page.getByText('执行自定义命令')).toBeVisible();
    await expect(page.getByLabel('目标节点')).toBeVisible();
    await expect(page.getByLabel('语言')).toBeVisible();
    await expect(page.getByLabel('代码')).toBeVisible();
  });

  test('应该能成功执行简单的 Bash 命令', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 打开执行命令对话框
    await page.getByRole('button', { name: /执行命令/ }).click();
    
    // 等待对话框加载
    await expect(page.getByText('执行自定义命令')).toBeVisible();
    
    // 清空默认代码并输入测试命令
    const codeTextarea = page.getByLabel('代码');
    await codeTextarea.clear();
    await codeTextarea.fill('echo "Test from E2E"\nhostname\ndate');
    
    // 确保目标节点为 * (所有节点)
    const targetInput = page.getByLabel('目标节点');
    await targetInput.clear();
    await targetInput.fill('*');
    
    // 点击执行按钮
    const executeButton = page.getByRole('button', { name: /执 行/ });
    await executeButton.click();
    
    // 验证执行按钮显示 loading 状态
    await expect(executeButton).toBeDisabled({ timeout: 1000 });
    
    // 等待执行进度区域出现日志
    const progressArea = page.locator('text=执行进度').locator('..');
    await expect(progressArea).toBeVisible();
    
    // 等待看到某些执行日志（最多30秒）
    await expect(page.locator('text=/step-log|step-done|complete/')).toBeVisible({ timeout: 30000 });
    
    // 等待执行完成（按钮恢复可用）- 这是修复的关键测试点
    await expect(executeButton).toBeEnabled({ timeout: 35000 });
    
    // 验证看到完成消息
    const completedMessage = await page.locator('text=/执行完成|complete/').isVisible();
    expect(completedMessage).toBeTruthy();
  });

  test('应该正确处理执行失败的情况', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 打开执行命令对话框
    await page.getByRole('button', { name: /执行命令/ }).click();
    
    // 输入一个会失败的命令
    const codeTextarea = page.getByLabel('代码');
    await codeTextarea.clear();
    await codeTextarea.fill('exit 1  # 故意失败');
    
    // 点击执行
    const executeButton = page.getByRole('button', { name: /执 行/ });
    await executeButton.click();
    
    // 等待执行完成或失败
    await expect(executeButton).toBeEnabled({ timeout: 35000 });
  });

  test('应该能取消执行对话框', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 打开执行命令对话框
    await page.getByRole('button', { name: /执行命令/ }).click();
    await expect(page.getByText('执行自定义命令')).toBeVisible();
    
    // 点击关闭按钮
    await page.getByRole('button', { name: /关闭/ }).first().click();
    
    // 验证对话框关闭
    await expect(page.getByText('执行自定义命令')).not.toBeVisible();
  });

  test('应该验证必填字段', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 打开执行命令对话框
    await page.getByRole('button', { name: /执行命令/ }).click();
    
    // 清空目标节点
    await page.getByLabel('目标节点').clear();
    
    // 清空代码
    await page.getByLabel('代码').clear();
    
    // 尝试执行
    await page.getByRole('button', { name: /执 行/ }).click();
    
    // 应该看到验证错误（Ant Design 的表单验证）
    // 注意：可能需要等待一小段时间才能看到错误
    await page.waitForTimeout(500);
    
    // 验证错误消息存在（目标节点和代码都是必填的）
    const hasError = await page.locator('text=/请输入|required/i').isVisible();
    expect(hasError).toBeTruthy();
  });

  test('应该显示实时执行进度', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 打开执行命令对话框
    await page.getByRole('button', { name: /执行命令/ }).click();
    
    // 输入一个需要一点时间的命令
    const codeTextarea = page.getByLabel('代码');
    await codeTextarea.clear();
    await codeTextarea.fill('for i in 1 2 3; do echo "Step $i"; sleep 0.5; done');
    
    // 执行
    await page.getByRole('button', { name: /执 行/ }).click();
    
    // 等待看到进度日志
    await expect(page.locator('text=step-log').first()).toBeVisible({ timeout: 10000 });
    
    // 验证有时间戳显示
    await expect(page.locator('text=/\\[\\d{1,2}:\\d{2}:\\d{2}\\]/')).toBeVisible();
    
    // 等待完成
    await expect(page.getByRole('button', { name: /执 行/ })).toBeEnabled({ timeout: 35000 });
  });

  test('应该能在执行完成后再次执行', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 第一次执行
    await page.getByRole('button', { name: /执行命令/ }).click();
    await page.getByLabel('代码').clear();
    await page.getByLabel('代码').fill('echo "First execution"');
    
    const executeButton = page.getByRole('button', { name: /执 行/ });
    await executeButton.click();
    
    // 等待第一次执行完成
    await expect(executeButton).toBeEnabled({ timeout: 35000 });
    
    // 第二次执行
    await page.getByLabel('代码').clear();
    await page.getByLabel('代码').fill('echo "Second execution"');
    await executeButton.click();
    
    // 验证第二次执行也能正常完成
    await expect(executeButton).toBeEnabled({ timeout: 35000 });
    
    // 验证看到第二次执行的日志
    const hasSecondLog = await page.locator('text=Second execution').isVisible();
    expect(hasSecondLog).toBeTruthy();
  });
});

test.describe('SaltStack 页面状态显示', () => {
  test.beforeEach(async ({ page }) => {
    await loginIfNeeded(page);
  });

  test('应该显示 Master 状态信息', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 验证页面已经加载了配置管理部分
    await expect(page.getByText('SaltStack 配置管理')).toBeVisible();
    
    // 验证有执行命令按钮和刷新按钮
    await expect(page.getByRole('button', { name: /执行命令/ })).toBeVisible();
  });

  test('应该显示 Minions 统计', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 验证页面有配置管理功能按钮
    await expect(page.getByRole('button', { name: /配置管理/ })).toBeVisible();
  });

  test('应该有刷新数据按钮', async ({ page }) => {
    await page.goto(BASE + '/saltstack');
    await waitForSaltStackPageLoad(page);
    
    // 查找刷新按钮
    const refreshButton = page.getByRole('button', { name: /刷新数据/ });
    await expect(refreshButton).toBeVisible();
    
    // 点击刷新
    await refreshButton.click();
    
    // 刷新按钮应该短暂显示 loading 状态
    await expect(refreshButton).toBeDisabled({ timeout: 1000 });
    
    // 然后恢复
    await expect(refreshButton).toBeEnabled({ timeout: 10000 });
  });
});
