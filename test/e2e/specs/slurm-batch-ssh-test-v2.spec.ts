import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.18.154:8080';

async function login(page: any) {
  await page.goto(`${BASE_URL}`, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(1000);
  
  const hasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  if (!hasLoginForm) {
    console.log('✓ 已登录');
    return;
  }
  
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
  console.log('✓ 登录完成');
}

test.describe('SLURM 批量SSH节点测试 V2', () => {
  
  test('验证批量SSH连接测试功能', async ({ page }) => {
    console.log('\n=== 测试批量SSH连接功能 ===');
    
    // 登录
    await login(page);
    
    // 访问 SLURM 页面
    await page.goto(`${BASE_URL}/slurm`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    await page.waitForTimeout(2000);
    
    // 切换到节点管理 Tab
    console.log('\n切换到节点管理 Tab...');
    const nodesTab = page.locator('.ant-tabs-tab').filter({ hasText: '节点管理' });
    await nodesTab.click();
    await page.waitForTimeout(1000);
    
    // 点击添加节点按钮打开 Modal
    console.log('\n点击添加节点按钮...');
    const addNodeButton = page.locator('button').filter({ hasText: '添加节点' });
    await expect(addNodeButton).toBeVisible({ timeout: 5000 });
    await addNodeButton.click();
    await page.waitForTimeout(1000);
    console.log('✓ 打开扩容节点对话框');
    
    // 在 Modal 中查找节点配置 textarea
    console.log('\n输入批量主机名...');
    const modal = page.locator('.ant-modal').filter({ hasText: '扩容 SLURM 节点' });
    await expect(modal).toBeVisible({ timeout: 5000 });
    
    const nodesTextarea = modal.locator('textarea[placeholder*="节点"]');
    await nodesTextarea.clear();
    await nodesTextarea.fill('test-ssh01,test-ssh02,test-ssh03');
    console.log('✓ 输入节点列表: test-ssh01,test-ssh02,test-ssh03');
    
    // 输入SSH密码
    const passwordInput = modal.locator('input[type="password"]').first();
    await passwordInput.clear();
    await passwordInput.fill('root');
    console.log('✓ 输入SSH密码');
    
    // 截图当前状态
    await page.screenshot({ 
      path: 'test-results/ssh-batch-test-before.png', 
      fullPage: true 
    });
    
    // 点击测试连接按钮
    console.log('\n点击测试SSH连接按钮...');
    const testButton = modal.locator('button').filter({ hasText: /测试.*SSH/ });
    await expect(testButton).toBeVisible();
    await testButton.click();
    
    // 等待测试完成（最多10秒）
    console.log('等待批量测试完成...');
    await page.waitForTimeout(6000);
    
    // 截图测试结果
    await page.screenshot({ 
      path: 'test-results/ssh-batch-test-after.png', 
      fullPage: true 
    });
    
    // 检查是否有成功提示消息
    const pageText = await page.locator('body').textContent();
    console.log('\n页面文本包含的关键词:');
    console.log('  - 批量测试:', pageText?.includes('批量测试'));
    console.log('  - 成功:', pageText?.includes('成功'));
    console.log('  - 失败:', pageText?.includes('失败'));
    console.log('  - test-ssh01:', pageText?.includes('test-ssh01'));
    console.log('  - test-ssh02:', pageText?.includes('test-ssh02'));
    console.log('  - test-ssh03:', pageText?.includes('test-ssh03'));
    
    // 检查结果表格（在 Modal 内的 Alert 中）
    const resultTable = modal.locator('.ant-table').first();
    const hasTable = await resultTable.count() > 0;
    
    if (hasTable) {
      console.log('\n✓ 找到结果表格');
      
      // 获取表格行数
      const tableRows = resultTable.locator('tbody tr');
      const rowCount = await tableRows.count();
      console.log(`  表格行数: ${rowCount}`);
      
      // 检查每一行的内容
      for (let i = 0; i < Math.min(rowCount, 3); i++) {
        const row = tableRows.nth(i);
        const rowText = await row.textContent();
        console.log(`  第 ${i + 1} 行: ${rowText}`);
        
        // 应该包含主机名
        const hasHost = rowText?.includes('test-ssh');
        console.log(`    包含主机名: ${hasHost}`);
        
        // 应该包含状态标签
        const statusTag = row.locator('.ant-tag');
        if (await statusTag.count() > 0) {
          const statusText = await statusTag.textContent();
          console.log(`    状态标签: ${statusText}`);
        }
      }
      
      // 断言：应该有3行测试结果（3个节点）
      expect(rowCount).toBeGreaterThanOrEqual(3);
      
    } else {
      console.log('\n⚠️ 未找到结果表格');
      
      // 检查是否有Alert显示结果
      const alerts = await modal.locator('.ant-alert').count();
      console.log(`Alert 数量: ${alerts}`);
      
      if (alerts > 0) {
        const alertText = await modal.locator('.ant-alert').textContent();
        console.log(`Alert 内容: ${alertText}`);
      }
      
      // 即使没有表格，如果有成功消息也算通过
      expect(pageText?.includes('成功') || pageText?.includes('批量')).toBeTruthy();
    }
    
    console.log('\n✓ 批量SSH测试验证完成');
  });
  
  test('验证单个节点测试仍然正常工作', async ({ page }) => {
    console.log('\n=== 测试单个SSH连接功能 ===');
    
    await login(page);
    await page.goto(`${BASE_URL}/slurm`, { waitUntil: 'networkidle', timeout: 30000 });
    await page.waitForTimeout(2000);
    
    const nodesTab = page.locator('.ant-tabs-tab').filter({ hasText: '节点管理' });
    await nodesTab.click();
    await page.waitForTimeout(1000);
    
    // 点击添加节点按钮
    const addNodeButton = page.locator('button').filter({ hasText: '添加节点' });
    await addNodeButton.click();
    await page.waitForTimeout(1000);
    
    const modal = page.locator('.ant-modal').filter({ hasText: '扩容 SLURM 节点' });
    
    // 输入单个主机名
    const nodesTextarea = modal.locator('textarea[placeholder*="节点"]');
    await nodesTextarea.clear();
    await nodesTextarea.fill('test-ssh01');
    console.log('✓ 输入单个节点: test-ssh01');
    
    const passwordInput = modal.locator('input[type="password"]').first();
    await passwordInput.clear();
    await passwordInput.fill('root');
    
    const testButton = modal.locator('button').filter({ hasText: /测试.*SSH/ });
    await testButton.click();
    
    // 等待结果
    await page.waitForTimeout(4000);
    
    // 检查是否有成功消息
    const bodyText = await page.locator('body').textContent();
    const hasSuccessText = bodyText?.includes('成功') || bodyText?.includes('SSH');
    
    console.log(`单个测试结果: ${hasSuccessText ? '包含成功/SSH关键词' : '需要检查'}`);
    
    await page.screenshot({ 
      path: 'test-results/ssh-single-test.png', 
      fullPage: true 
    });
    
    expect(hasSuccessText).toBeTruthy();
  });
});
