const { test, expect } = require('@playwright/test');

/**
 * SaltStack 命令执行器快速测试
 * 仅测试核心功能
 */

test.describe('SaltStack 命令执行器 - 快速测试', () => {
  test('执行命令并验证输出', async ({ page }) => {
    // 登录
    await page.goto('/login');
    // 等待登录表单加载
    await page.waitForSelector('input[type="text"]', { timeout: 10000 });
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    // 等待跳转到 projects 页面
    await page.waitForURL('**/projects', { timeout: 15000 });
    
    // 直接导航到 SLURM 页面
    await page.goto('/slurm');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);
    
    // 切换到 SaltStack 集成 tab
    await page.click('text=SaltStack 集成');
    await page.waitForTimeout(2000);
    
    // 等待命令执行表单加载
    await page.waitForSelector('button:has-text("执行命令")', { timeout: 10000 });
    
    // 执行 test.ping 命令（更快）
    // 目标节点已经默认选择"所有节点"，不需要修改
    
    // 找到所有 Select 组件，第二个是 Salt 函数选择器
    const selects = await page.locator('.ant-select').all();
    if (selects.length >= 2) {
      await selects[1].click();
      await page.waitForTimeout(500);
      // 选择 test.ping
      await page.click('.ant-select-item:has-text("test.ping")');
    }
    
    // 点击执行命令按钮
    await page.click('button:has-text("执行命令")');
    
    // 等待执行完成（增加超时时间）
    await page.waitForSelector('text=最新执行结果', { timeout: 60000 });
    
    // 验证成功标签（使用 first() 避免 strict mode 错误）
    const successTag = page.locator('.ant-tag-success:has-text("成功")').first();
    await expect(successTag).toBeVisible({ timeout: 5000 });
    
    // 验证输出不为空
    const outputPre = await page.locator('pre').first();
    const outputText = await outputPre.textContent();
    
    console.log('✅ 命令执行成功');
    console.log('📦 输出内容:', outputText);
    
    // 验证输出不为空（Salt API 可能返回空结果，但结构应该正确）
    expect(outputText.length).toBeGreaterThan(10);
    expect(outputText).toContain('success');
    
    // 测试复制功能 - 只验证按钮可点击，不验证剪贴板内容（在 headless 模式下可能不可用）
    const copyButton = page.locator('button:has-text("复制输出")');
    await expect(copyButton).toBeVisible();
    await copyButton.click();
    await page.waitForTimeout(500);
    
    console.log('✅ 所有测试通过！');
    console.log('✅ 1. 命令执行成功');
    console.log('✅ 2. 输出格式正确');
    console.log('✅ 3. 复制按钮可用');
  });
});
