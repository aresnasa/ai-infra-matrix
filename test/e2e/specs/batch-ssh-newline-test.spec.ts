import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.18.154:8080';
const SSH_PASSWORD = 'rootpass123';

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

test.describe('批量SSH测试 - 换行符分隔支持', () => {
  
  test('测试换行符分隔的批量主机', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试换行符分隔的批量SSH连接');
    console.log('========================================\n');
    
    await login(page);
    
    await page.goto(`${BASE_URL}/slurm`, { waitUntil: 'networkidle', timeout: 30000 });
    await page.waitForTimeout(2000);
    
    const nodesTab = page.locator('.ant-tabs-tab').filter({ hasText: '节点管理' });
    await nodesTab.click();
    await page.waitForTimeout(1000);
    console.log('✓ 导航到节点管理页面');
    
    const addNodeButton = page.locator('button').filter({ hasText: '添加节点' });
    await addNodeButton.click();
    await page.waitForTimeout(1000);
    console.log('✓ 打开扩容节点对话框');
    
    const modal = page.locator('.ant-modal').filter({ hasText: '扩容 SLURM 节点' });
    
    // 输入换行符分隔的节点列表
    const nodesTextarea = modal.locator('textarea[placeholder*="节点"]');
    await nodesTextarea.clear();
    await nodesTextarea.fill('test-ssh01\ntest-ssh02\ntest-ssh03');
    console.log('✓ 输入3个节点（换行符分隔）');
    
    const passwordInput = modal.locator('input[type="password"]').first();
    await passwordInput.clear();
    await passwordInput.fill(SSH_PASSWORD);
    console.log(`✓ 输入SSH密码`);
    
    await page.screenshot({ 
      path: 'test-results/newline-batch-test-ready.png', 
      fullPage: true 
    });
    
    console.log('\n开始批量SSH连接测试...');
    const testButton = modal.locator('button').filter({ hasText: /测试.*SSH/ });
    await testButton.click();
    
    console.log('等待测试完成（最多10秒）...');
    await page.waitForTimeout(10000);
    
    await page.screenshot({ 
      path: 'test-results/newline-batch-test-complete.png', 
      fullPage: true 
    });
    
    console.log('\n========== 测试结果分析 ==========');
    
    const bodyText = await page.locator('body').textContent() || '';
    
    console.log('\n关键词检测:');
    console.log(`  ✓ 批量测试: ${bodyText.includes('批量测试')}`);
    console.log(`  ✓ test-ssh01: ${bodyText.includes('test-ssh01')}`);
    console.log(`  ✓ test-ssh02: ${bodyText.includes('test-ssh02')}`);
    console.log(`  ✓ test-ssh03: ${bodyText.includes('test-ssh03')}`);
    
    const resultTable = modal.locator('.ant-table');
    const hasTable = await resultTable.count() > 0;
    
    if (hasTable) {
      console.log('\n✅ 找到测试结果表格');
      
      const tableRows = resultTable.locator('tbody tr');
      const rowCount = await tableRows.count();
      console.log(`   表格行数: ${rowCount}`);
      
      console.log('\n节点测试详情:');
      for (let i = 0; i < rowCount; i++) {
        const row = tableRows.nth(i);
        const cells = row.locator('td');
        const cellCount = await cells.count();
        
        if (cellCount >= 4) {
          const host = await cells.nth(0).textContent();
          const status = await cells.nth(1).textContent();
          const duration = await cells.nth(2).textContent();
          
          console.log(`   节点 ${i + 1}: ${host} - ${status} - ${duration}`);
        }
      }
      
      expect(rowCount).toBe(3);
      console.log('\n✅ 验证通过: 换行符分隔成功识别3个节点');
      
    } else {
      console.log('\n⚠️  未找到结果表格');
    }
    
    console.log('\n========================================');
    
    expect(bodyText).toContain('test-ssh01');
    expect(bodyText).toContain('test-ssh02');
    expect(bodyText).toContain('test-ssh03');
  });
  
  test('测试混合分隔符（逗号+换行）', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试混合分隔符（逗号+换行）');
    console.log('========================================\n');
    
    await login(page);
    
    await page.goto(`${BASE_URL}/slurm`, { waitUntil: 'networkidle', timeout: 30000 });
    await page.waitForTimeout(2000);
    
    const nodesTab = page.locator('.ant-tabs-tab').filter({ hasText: '节点管理' });
    await nodesTab.click();
    await page.waitForTimeout(1000);
    
    const addNodeButton = page.locator('button').filter({ hasText: '添加节点' });
    await addNodeButton.click();
    await page.waitForTimeout(1000);
    
    const modal = page.locator('.ant-modal').filter({ hasText: '扩容 SLURM 节点' });
    
    // 混合使用逗号和换行符
    const nodesTextarea = modal.locator('textarea[placeholder*="节点"]');
    await nodesTextarea.clear();
    await nodesTextarea.fill('test-ssh01,test-ssh02\ntest-ssh03');
    console.log('✓ 输入混合分隔符：test-ssh01,test-ssh02\\ntest-ssh03');
    
    const passwordInput = modal.locator('input[type="password"]').first();
    await passwordInput.clear();
    await passwordInput.fill(SSH_PASSWORD);
    
    const testButton = modal.locator('button').filter({ hasText: /测试.*SSH/ });
    await testButton.click();
    
    await page.waitForTimeout(10000);
    
    await page.screenshot({ 
      path: 'test-results/mixed-separator-test.png', 
      fullPage: true 
    });
    
    const bodyText = await page.locator('body').textContent() || '';
    
    console.log('\n关键词检测:');
    console.log(`  ✓ test-ssh01: ${bodyText.includes('test-ssh01')}`);
    console.log(`  ✓ test-ssh02: ${bodyText.includes('test-ssh02')}`);
    console.log(`  ✓ test-ssh03: ${bodyText.includes('test-ssh03')}`);
    
    const resultTable = modal.locator('.ant-table');
    const hasTable = await resultTable.count() > 0;
    
    if (hasTable) {
      const tableRows = resultTable.locator('tbody tr');
      const rowCount = await tableRows.count();
      console.log(`\n表格行数: ${rowCount}`);
      
      expect(rowCount).toBe(3);
      console.log('✅ 验证通过: 混合分隔符成功识别3个节点');
    }
    
    console.log('\n========================================');
    
    expect(bodyText).toContain('test-ssh01');
    expect(bodyText).toContain('test-ssh02');
    expect(bodyText).toContain('test-ssh03');
  });
});
