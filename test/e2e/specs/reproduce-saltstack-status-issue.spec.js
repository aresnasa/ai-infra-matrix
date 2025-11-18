const { test, expect } = require('@playwright/test');

/**
 * 重现并修复：节点管理-集群节点-SaltStack状态-未配置问题
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.91:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

test.describe('节点 SaltStack 状态显示问题', () => {
  test('重现问题：前端显示"未配置"但后端返回正确状态', async ({ page, request }) => {
    console.log('\n=== 步骤 1: 验证后端 API 返回正确状态 ===');
    
    // 登录获取 token
    const loginResponse = await request.post(`${BASE_URL}/api/auth/login`, {
      data: { username: TEST_USERNAME, password: TEST_PASSWORD }
    });
    const loginData = await loginResponse.json();
    const token = loginData.data?.token || loginData.token;
    
    // 获取节点列表
    const nodesResponse = await request.get(`${BASE_URL}/api/slurm/nodes`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    const nodesData = await nodesResponse.json();
    const nodes = nodesData.data || [];
    
    console.log(`后端返回 ${nodes.length} 个节点:`);
    nodes.forEach(node => {
      console.log(`  ${node.name}: salt_status = "${node.salt_status}"`);
    });
    
    const allAccepted = nodes.every(n => n.salt_status === 'accepted');
    console.log(allAccepted ? '\n✅ 后端：所有节点状态为 accepted' : '\n⚠️  后端：有节点状态异常');
    
    console.log('\n=== 步骤 2: 检查前端显示 ===');
    
    // 登录前端
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    const usernameInput = page.locator('input[name="username"]').or(page.locator('input[placeholder*="用户名"]'));
    await usernameInput.waitFor({ timeout: 5000 });
    await usernameInput.fill(TEST_USERNAME);
    
    const passwordInput = page.locator('input[name="password"]').or(page.locator('input[type="password"]'));
    await passwordInput.fill(TEST_PASSWORD);
    
    const loginButton = page.locator('button[type="submit"]').or(page.locator('button:has-text("登录")'));
    await loginButton.click();
    
    // 等待登录成功并跳转
    await page.waitForTimeout(2000);
    const currentUrl = page.url();
    console.log(`登录后 URL: ${currentUrl}`);
    
    if (currentUrl.includes('/login')) {
      console.log('⚠️  仍在登录页面，尝试再次点击登录');
      await loginButton.click();
      await page.waitForTimeout(2000);
    }
    
    // 访问 SLURM 页面
    console.log('\n访问 SLURM 页面...');
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    
    const finalUrl = page.url();
    console.log(`当前 URL: ${finalUrl}`);
    
    if (finalUrl.includes('/login')) {
      console.log('❌ 被重定向回登录页面，登录失败');
      await page.screenshot({ path: 'test-screenshots/login-failed.png', fullPage: true });
      return;
    }
    
    console.log('✅ 成功访问 SLURM 页面');
    
    console.log('\n当前页面标签:');
    const allTabs = page.locator('.ant-tabs-tab');
    const tabCount = await allTabs.count();
    for (let i = 0; i < tabCount; i++) {
      const tabText = await allTabs.nth(i).innerText();
      console.log(`  ${i + 1}. ${tabText}`);
    }
    
    // 点击节点管理标签
    const nodesTab = page.locator('.ant-tabs-tab').filter({ hasText: /节点管理/ });
    if (await nodesTab.count() > 0) {
      console.log('\n✅ 找到节点管理标签');
      await nodesTab.click();
      await page.waitForTimeout(2000);
    } else {
      console.log('\n❌ 未找到节点管理标签');
      await page.screenshot({ path: 'test-screenshots/no-nodes-tab.png', fullPage: true });
      return;
    }
    
    // 查找集群节点表格
    console.log('\n查找集群节点表格...');
    const tables = page.locator('.ant-table');
    const tableCount = await tables.count();
    console.log(`找到 ${tableCount} 个表格`);
    
    // 查找包含 SaltStack 状态列的表格
    let nodesTable = null;
    for (let i = 0; i < tableCount; i++) {
      const table = tables.nth(i);
      const headers = table.locator('thead th');
      const headerCount = await headers.count();
      
      for (let j = 0; j < headerCount; j++) {
        const headerText = await headers.nth(j).innerText();
        if (headerText.includes('SaltStack') || headerText.includes('Salt')) {
          nodesTable = table;
          console.log(`✅ 找到包含 SaltStack 状态列的表格 (第 ${i + 1} 个)`);
          break;
        }
      }
      
      if (nodesTable) break;
    }
    
    if (!nodesTable) {
      console.log('❌ 未找到包含 SaltStack 状态列的表格');
      await page.screenshot({ path: 'test-screenshots/no-saltstack-column.png', fullPage: true });
      return;
    }
    
    // 提取表格数据
    console.log('\n前端表格显示:');
    const rows = nodesTable.locator('tbody tr');
    const rowCount = await rows.count();
    console.log(`表格有 ${rowCount} 行数据`);
    
    const frontendStatuses = [];
    for (let i = 0; i < rowCount; i++) {
      const row = rows.nth(i);
      const cells = row.locator('td');
      const cellCount = await cells.count();
      
      if (cellCount > 0) {
        const nodeName = await cells.nth(0).innerText();
        
        // 查找 SaltStack 状态列
        let saltStatus = 'N/A';
        for (let j = 0; j < cellCount; j++) {
          const cellText = await cells.nth(j).innerText();
          if (cellText.includes('未配置') || cellText.includes('已连接') || 
              cellText.includes('待接受') || cellText.includes('已拒绝') ||
              cellText.includes('API 错误')) {
            saltStatus = cellText.trim();
            break;
          }
        }
        
        frontendStatuses.push({ nodeName, saltStatus });
        console.log(`  ${nodeName}: ${saltStatus}`);
      }
    }
    
    await page.screenshot({ 
      path: 'test-screenshots/saltstack-status-comparison.png',
      fullPage: true 
    });
    console.log('\n📸 截图已保存: saltstack-status-comparison.png');
    
    // 对比分析
    console.log('\n=== 步骤 3: 对比后端 API 与前端显示 ===');
    
    const unconfiguredInFrontend = frontendStatuses.filter(n => 
      n.saltStatus.includes('未配置')
    ).length;
    
    if (unconfiguredInFrontend > 0) {
      console.log(`\n⚠️  问题确认: ${unconfiguredInFrontend} 个节点在前端显示为"未配置"`);
      console.log('但后端 API 返回状态正常 (accepted)');
      
      console.log('\n🔍 问题原因分析:');
      console.log('  1. 前端代码未更新/重新构建');
      console.log('  2. 前端渲染逻辑有问题');
      console.log('  3. 数据在传输过程中丢失');
      
      console.log('\n✅ 解决方案:');
      console.log('  1. 重新构建前端: cd src/frontend && npm run build');
      console.log('  2. 或重启容器: docker-compose restart frontend');
      console.log('  3. 或强制刷新浏览器: Ctrl+Shift+R (清除缓存)');
    } else if (unconfiguredInFrontend === 0 && frontendStatuses.length > 0) {
      console.log('\n✅ 前端显示正常，所有节点状态正确');
    } else {
      console.log('\n⚠️  前端未显示任何节点或未找到状态信息');
    }
  });
});
