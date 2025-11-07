const { test, expect } = require('@playwright/test');

/**
 * SLURM 页面手动检查 - 调试用
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test.describe('SLURM 页面调试', () => {
  test('手动检查页面元素', async ({ page }) => {
    console.log('\n🔍 手动检查 SLURM 页面...');
    
    // 1. 先登录
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    // 登录
    const usernameInput = page.locator('input[type="text"]').first();
    const passwordInput = page.locator('input[type="password"]').first();
    const loginButton = page.locator('button[type="submit"]').first();
    
    await usernameInput.fill('admin');
    await passwordInput.fill('admin123');
    await loginButton.click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    console.log('✅ 登录完成');
    
    // 2. 访问 SLURM 页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    
    // 3. 截图
    await page.screenshot({ 
      path: 'test-screenshots/slurm-manual-check-full.png',
      fullPage: true 
    });
    console.log('📸 截图已保存');
    
    // 4. 输出页面标题
    const title = await page.title();
    console.log(`页面标题: ${title}`);
    
    // 5. 查找所有标题元素
    console.log('\n查找标题元素:');
    const h1Elements = await page.locator('h1').allTextContents();
    const h2Elements = await page.locator('h2').allTextContents();
    const h3Elements = await page.locator('h3').allTextContents();
    
    if (h1Elements.length > 0) {
      console.log('  H1:', h1Elements);
    }
    if (h2Elements.length > 0) {
      console.log('  H2:', h2Elements);
    }
    if (h3Elements.length > 0) {
      console.log('  H3:', h3Elements);
    }
    
    // 6. 查找表格
    console.log('\n查找表格:');
    const tables = await page.locator('table').count();
    console.log(`  表格数量: ${tables}`);
    
    if (tables > 0) {
      for (let i = 0; i < tables; i++) {
        const table = page.locator('table').nth(i);
        const headers = await table.locator('th').allTextContents();
        console.log(`  表格 ${i + 1} 表头:`, headers);
      }
    }
    
    // 7. 查找所有按钮
    console.log('\n查找按钮:');
    const buttons = await page.locator('button').allTextContents();
    console.log(`  按钮文本:`, buttons.filter(b => b.trim()));
    
    // 8. 检查 API 响应
    console.log('\n检查 API 响应:');
    
    // 获取 localStorage 中的 token
    const token = await page.evaluate(() => localStorage.getItem('token'));
    console.log(`  Token: ${token ? token.substring(0, 20) + '...' : '无'}`);
    
    // 直接在浏览器中调用 API
    const apiResults = await page.evaluate(async (baseURL, token) => {
      const results = {};
      
      const headers = {
        'Content-Type': 'application/json'
      };
      
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      
      try {
        // 测试 summary
        const summaryResp = await fetch(`${baseURL}/api/slurm/summary`, { headers });
        results.summary = {
          status: summaryResp.status,
          ok: summaryResp.ok,
          data: summaryResp.ok ? await summaryResp.json() : await summaryResp.text()
        };
      } catch (e) {
        results.summary = { error: e.message };
      }
      
      try {
        // 测试 nodes
        const nodesResp = await fetch(`${baseURL}/api/slurm/nodes`, { headers });
        results.nodes = {
          status: nodesResp.status,
          ok: nodesResp.ok,
          data: nodesResp.ok ? await nodesResp.json() : await nodesResp.text()
        };
      } catch (e) {
        results.nodes = { error: e.message };
      }
      
      return results;
    }, BASE_URL, token);
    
    console.log('\n  API 测试结果:');
    console.log('  Summary API:', apiResults.summary.status, apiResults.summary.ok ? '✅' : '❌');
    if (!apiResults.summary.ok) {
      console.log('    错误:', apiResults.summary.data);
    } else if (apiResults.summary.data.demo) {
      console.log('    ⚠️  返回演示数据');
    }
    
    console.log('  Nodes API:', apiResults.nodes.status, apiResults.nodes.ok ? '✅' : '❌');
    if (!apiResults.nodes.ok) {
      console.log('    错误:', apiResults.nodes.data);
    } else {
      const nodesData = apiResults.nodes.data;
      const nodes = nodesData.data?.data || nodesData.data || [];
      console.log(`    节点数量: ${nodes.length}`);
      if (nodes.length > 0) {
        console.log(`    第一个节点:`, nodes[0]);
      }
    }
    
    // 9. 等待一段时间以便手动检查
    console.log('\n⏸️  浏览器将保持打开 30 秒，可以手动检查...');
    await page.waitForTimeout(30000);
  });
});
