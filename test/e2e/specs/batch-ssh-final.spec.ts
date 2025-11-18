import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.18.154:8080';
const SSH_PASSWORD = 'rootpass123'; // test-containers 的root密码

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

test.describe('SLURM 批量SSH节点测试（最终版）', () => {
  
  test('批量SSH测试: test-ssh01, test-ssh02, test-ssh03', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试批量SSH连接功能');
    console.log('节点: test-ssh01, test-ssh02, test-ssh03');
    console.log('========================================\n');
    
    await login(page);
    
    // 访问 SLURM 页面
    await page.goto(`${BASE_URL}/slurm`, { waitUntil: 'networkidle', timeout: 30000 });
    await page.waitForTimeout(2000);
    
    // 切换到节点管理 Tab
    const nodesTab = page.locator('.ant-tabs-tab').filter({ hasText: '节点管理' });
    await nodesTab.click();
    await page.waitForTimeout(1000);
    console.log('✓ 导航到节点管理页面');
    
    // 点击添加节点按钮
    const addNodeButton = page.locator('button').filter({ hasText: '添加节点' });
    await addNodeButton.click();
    await page.waitForTimeout(1000);
    console.log('✓ 打开扩容节点对话框');
    
    const modal = page.locator('.ant-modal').filter({ hasText: '扩容 SLURM 节点' });
    
    // 输入批量节点
    const nodesTextarea = modal.locator('textarea[placeholder*="节点"]');
    await nodesTextarea.clear();
    await nodesTextarea.fill('test-ssh01,test-ssh02,test-ssh03');
    console.log('✓ 输入3个节点（逗号分隔）');
    
    // 输入正确的SSH密码
    const passwordInput = modal.locator('input[type="password"]').first();
    await passwordInput.clear();
    await passwordInput.fill(SSH_PASSWORD);
    console.log(`✓ 输入SSH密码: ${SSH_PASSWORD}`);
    
    // 截图测试前状态
    await page.screenshot({ 
      path: 'test-results/batch-ssh-test-ready.png', 
      fullPage: true 
    });
    
    // 点击测试按钮
    console.log('\n开始批量SSH连接测试...');
    const testButton = modal.locator('button').filter({ hasText: /测试.*SSH/ });
    await testButton.click();
    
    // 等待测试完成
    console.log('等待测试完成（最多10秒）...');
    await page.waitForTimeout(10000);
    
    // 截图测试后状态
    await page.screenshot({ 
      path: 'test-results/batch-ssh-test-complete.png', 
      fullPage: true 
    });
    
    // 分析测试结果
    console.log('\n========== 测试结果分析 ==========');
    
    const bodyText = await page.locator('body').textContent() || '';
    
    console.log('\n关键词检测:');
    console.log(`  ✓ 批量测试: ${bodyText.includes('批量测试')}`);
    console.log(`  ✓ test-ssh01: ${bodyText.includes('test-ssh01')}`);
    console.log(`  ✓ test-ssh02: ${bodyText.includes('test-ssh02')}`);
    console.log(`  ✓ test-ssh03: ${bodyText.includes('test-ssh03')}`);
    console.log(`  ✓ 包含"成功": ${bodyText.includes('成功')}`);
    
    // 检查结果表格
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
          const message = await cells.nth(3).textContent();
          
          console.log(`\n   节点 ${i + 1}:`);
          console.log(`     主机: ${host}`);
          console.log(`     状态: ${status}`);
          console.log(`     耗时: ${duration}`);
          console.log(`     消息: ${message?.substring(0, 60)}...`);
        }
      }
      
      // 断言：应该有3行（3个节点）
      expect(rowCount).toBe(3);
      console.log('\n✅ 验证通过: 表格包含3个节点的测试结果');
      
    } else {
      console.log('\n⚠️  未找到结果表格');
      
      // 检查Alert
      const alerts = modal.locator('.ant-alert');
      const alertCount = await alerts.count();
      console.log(`   Alert数量: ${alertCount}`);
      
      for (let i = 0; i < alertCount; i++) {
        const alertType = await alerts.nth(i).getAttribute('class');
        const alertText = await alerts.nth(i).textContent();
        console.log(`   Alert ${i + 1} (${alertType?.includes('success') ? '成功' : alertType?.includes('error') ? '错误' : '其他'}): `);
        console.log(`     ${alertText?.substring(0, 100)}...`);
      }
    }
    
    console.log('\n========================================');
    console.log('测试完成');
    console.log('========================================\n');
    
    // 基本断言
    expect(bodyText).toContain('test-ssh01');
    expect(bodyText).toContain('test-ssh02');
    expect(bodyText).toContain('test-ssh03');
  });
});
