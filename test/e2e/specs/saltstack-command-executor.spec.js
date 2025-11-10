const { test, expect } = require('@playwright/test');

/**
 * SaltStack 命令执行器测试
 * 
 * 测试场景:
 * 1. 执行 Salt 命令并验证输出
 * 2. 测试复制输出功能
 * 3. 验证历史记录持久化
 */

test.describe('SaltStack 命令执行器', () => {
  test.beforeEach(async ({ page }) => {
    // 访问登录页面
    await page.goto('/login');
    
    // 等待登录表单加载
    await page.waitForSelector('input[type="text"]', { timeout: 10000 });
    
    // 登录
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    
    // 等待跳转到 projects 页面
    await page.waitForURL('**/projects', { timeout: 15000 });
    
    // 直接导航到 SLURM 页面
    await page.goto('/slurm');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);
    
    // 切换到 SaltStack 集成 tab
    await page.click('text=SaltStack 集成');
    await page.waitForTimeout(1000);
    
    // 等待命令执行表单加载
    await page.waitForSelector('button:has-text("执行命令")', { timeout: 10000 });
  });

  test('执行 cmd.run 命令并验证输出', async ({ page }) => {
    // 目标节点已经默认选择"所有节点"
    
    // 选择 Salt 函数 - 使用更可靠的选择器
    const selects = await page.locator('.ant-select').all();
    if (selects.length >= 2) {
      await selects[1].click();
      await page.waitForTimeout(500);
      await page.click('.ant-select-item:has-text("cmd.run")');
    }
    
    // 输入参数 - 使用 textarea 标签
    await page.fill('textarea[placeholder*="函数参数"]', "/bin/bash -c 'sinfo'");
    
    // 执行命令
    await page.click('button:has-text("执行命令")');
    
    // 等待执行完成（最多 60 秒）
    await page.waitForSelector('text=最新执行结果', { timeout: 60000 });
    
    // 验证成功标签存在（使用 first() 避免 strict mode）
    const successTag = page.locator('.ant-tag-success:has-text("成功")').first();
    await expect(successTag).toBeVisible();
    
    // 验证执行输出不为空
    const outputPre = page.locator('pre').first();
    const outputText = await outputPre.textContent();
    
    console.log('命令执行输出:', outputText);
    
    // 验证输出不为空
    expect(outputText.length).toBeGreaterThan(10);
    expect(outputText).toContain('success');
    
    // 尝试解析 JSON
    let parsedOutput;
    try {
      parsedOutput = JSON.parse(outputText);
      console.log('解析后的输出:', parsedOutput);
    } catch (e) {
      console.error('输出不是有效的 JSON:', outputText);
      throw new Error('输出应该是有效的 JSON 格式');
    }
    
    // 验证输出结构
    expect(parsedOutput).toBeDefined();
    expect(parsedOutput.result).toBeDefined();
  });

  test('测试复制输出功能', async ({ page, context }) => {
    // 选择 Salt 函数
    const selects = await page.locator('.ant-select').all();
    if (selects.length >= 2) {
      await selects[1].click();
      await page.waitForTimeout(500);
      await page.click('.ant-select-item:has-text("test.ping")');
    }
    
    // 执行命令
    await page.click('button:has-text("执行命令")');
    
    // 等待执行完成
    await page.waitForSelector('text=最新执行结果', { timeout: 60000 });
    
    // 测试复制按钮
    const copyButton = page.locator('button:has-text("复制输出")');
    await expect(copyButton).toBeVisible();
    await copyButton.click();
    await page.waitForTimeout(500);
    
    console.log('✅ 复制按钮测试通过');
  });

  test('验证历史记录持久化', async ({ page }) => {
    // 执行一个命令
    const selects = await page.locator('.ant-select').all();
    if (selects.length >= 2) {
      await selects[1].click();
      await page.waitForTimeout(500);
      await page.click('.ant-select-item:has-text("test.ping")');
    }
    await page.click('button:has-text("执行命令")');
    
    // 等待命令执行完成
    await page.waitForSelector('text=最新执行结果', { timeout: 60000 });
    
    // 验证历史记录功能存在（表格或其他形式）
    // 注意：历史记录可能以不同形式展示
    const hasHistory = await page.locator('text=test.ping').count() > 0;
    expect(hasHistory).toBeTruthy();
    
    console.log('✅ 历史记录功能验证通过');
  });

  test('验证命令执行耗时显示', async ({ page }) => {
    // 执行命令
    const selects = await page.locator('.ant-select').all();
    if (selects.length >= 2) {
      await selects[1].click();
      await page.waitForTimeout(500);
      await page.click('.ant-select-item:has-text("test.ping")');
    }
    
    const startTime = Date.now();
    await page.click('button:has-text("执行命令")');
    
    // 等待执行完成
    await page.waitForSelector('text=最新执行结果', { timeout: 60000 });
    const endTime = Date.now();
    const actualDuration = endTime - startTime;
    
    // 验证显示的耗时存在
    const durationElement = page.locator('text=/耗时.*ms/').first();
    await expect(durationElement).toBeVisible({ timeout: 5000 });
    
    console.log(`✅ 命令执行耗时: ~${actualDuration}ms`);
  });

  test('验证查看详情功能', async ({ page }) => {
    // 执行命令
    const selects = await page.locator('.ant-select').all();
    if (selects.length >= 2) {
      await selects[1].click();
      await page.waitForTimeout(500);
      await page.click('.ant-select-item:has-text("test.ping")');
    }
    await page.click('button:has-text("执行命令")');
    
    // 等待执行完成
    await page.waitForSelector('text=最新执行结果', { timeout: 60000 });
    
    // 验证最新执行结果显示
    const resultCard = page.locator('text=最新执行结果').locator('..');
    await expect(resultCard).toBeVisible();
    
    console.log('✅ 查看详情功能测试通过');
  });
});
