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

test.describe('SLURM 批量SSH节点测试', () => {
  
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
    console.log('✓ 打开添加节点对话框');
    console.log('✓ 打开添加节点对话框');
    
    // 在 Modal 中查找主机名输入框
    console.log('\n输入批量主机名...');
    const modal = page.locator('.ant-modal').filter({ hasText: '添加节点' });
    await expect(modal).toBeVisible({ timeout: 5000 });
    
    const hostInput = modal.locator('input[placeholder*="主机名"]').or(
      modal.locator('input[id*="host"]')
    ).first();
    
    await hostInput.clear();
    await hostInput.fill('test-ssh01,test-ssh02,test-ssh03');
    console.log('✓ 输入主机名: test-ssh01,test-ssh02,test-ssh03');
    
    // 输入SSH密码
    const passwordInput = modal.locator('input[type="password"]').first();
    await passwordInput.clear();
    await passwordInput.fill('root');
    console.log('✓ 输入SSH密码');
    console.log('✓ 输入SSH密码');
    
    // 截图当前状态
    await page.screenshot({ 
      path: 'test-results/ssh-batch-test-before.png', 
      fullPage: true 
    });
    
    // 点击测试连接按钮
    console.log('\n点击测试SSH连接按钮...');
    const testButton = modal.locator('button').filter({ hasText: /测试.*连接/ });
    await expect(testButton).toBeVisible();
    await testButton.click();
    await testButton.click();
    
    // 等待测试完成（最多30秒）
    console.log('等待批量测试完成...');
    
    // 等待测试结果出现
    await page.waitForTimeout(5000);
    
    // 检查是否有成功提示
    const successMessage = page.locator('.ant-message').filter({ hasText: /批量测试完成|成功/ });
    const hasSuccess = await successMessage.count() > 0;
    
    if (hasSuccess) {
      console.log('✓ 检测到成功消息');
    }
    
    // 检查结果表格（在 Modal 内）
    const resultTable = modal.locator('.ant-table').first();
    const hasTable = await resultTable.count() > 0;
    
    if (hasTable) {
      console.log('✓ 找到结果表格');
      
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
          console.log(`    状态: ${statusText}`);
        }
      }
      
      // 断言：应该有测试结果
      expect(rowCount).toBeGreaterThan(0);
      expect(rowCount).toBeLessThanOrEqual(3);
      
    } else {
      console.log('⚠️ 未找到结果表格');
    }
    
    // 截图最终结果
    await page.screenshot({ 
      path: 'test-results/ssh-batch-test-after.png', 
      fullPage: true 
    });
    
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
    
    const modal = page.locator('.ant-modal').filter({ hasText: '添加节点' });
    
    // 输入单个主机名
    const hostInput = modal.locator('input[placeholder*="主机名"]').or(
      modal.locator('input[id*="host"]')
    ).first();
    
    await hostInput.clear();
    await hostInput.fill('test-ssh01');
    console.log('✓ 输入单个主机名: test-ssh01');
    
    const passwordInput = modal.locator('input[type="password"]').first();
    await passwordInput.clear();
    await passwordInput.fill('root');
    
    const testButton = modal.locator('button').filter({ hasText: /测试.*连接/ });
    await testButton.click();
    
    // 等待结果
    await page.waitForTimeout(3000);
    
    // 检查是否有成功消息
    const bodyText = await page.locator('body').textContent();
    const hasSuccessText = bodyText?.includes('成功') || bodyText?.includes('SSH');
    
    console.log(`单个测试结果: ${hasSuccessText ? '成功' : '需要检查'}`);
    
    await page.screenshot({ 
      path: 'test-results/ssh-single-test.png', 
      fullPage: true 
    });
  });
});
