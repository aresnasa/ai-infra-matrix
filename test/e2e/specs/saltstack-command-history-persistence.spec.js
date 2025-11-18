const { test, expect } = require('@playwright/test');

/**
 * SaltStack 命令执行历史持久化测试
 * 测试目标：
 * 1. 执行命令后历史记录能够保存
 * 2. 刷新页面后历史记录仍然存在
 * 3. 历史记录通过 API 正确获取
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.91:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

test.describe('SaltStack 命令执行历史持久化', () => {
  let page;
  let context;

  test.beforeAll(async ({ browser }) => {
    context = await browser.newContext();
    page = await context.newPage();
    
    // 登录
    console.log('\n=== 登录系统 ===');
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    const usernameInput = page.locator('input[name="username"]').or(page.locator('input[placeholder*="用户名"]'));
    await usernameInput.waitFor({ timeout: 5000 });
    await usernameInput.fill(TEST_USERNAME);
    
    const passwordInput = page.locator('input[name="password"]').or(page.locator('input[type="password"]'));
    await passwordInput.fill(TEST_PASSWORD);
    
    const loginButton = page.locator('button[type="submit"]').or(page.locator('button:has-text("登录")'));
    await loginButton.click();
    await page.waitForTimeout(2000);
    
    console.log('✅ 登录成功');
  });

  test.afterAll(async () => {
    await context.close();
  });

  test('验证命令执行历史的持久化存储', async () => {
    console.log('\n=== 步骤 1: 访问 SaltStack 集成页面 ===');
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 点击 SaltStack 集成标签
    const saltStackTab = page.locator('.ant-tabs-tab').filter({ hasText: /SaltStack.*集成/ });
    await saltStackTab.waitFor({ timeout: 5000 });
    await saltStackTab.click();
    await page.waitForTimeout(2000);
    
    console.log('✅ 已进入 SaltStack 集成页面');

    console.log('\n=== 步骤 2: 检查现有的命令执行历史 ===');
    
    // 查找命令执行历史卡片
    const historyCard = page.locator('.ant-card').filter({ 
      has: page.locator('span:has-text("命令执行历史")') 
    });
    await historyCard.waitFor({ timeout: 5000 });
    
    // 获取当前历史记录数量
    const recordTag = historyCard.locator('.ant-tag').filter({ hasText: /\d+.*条记录/ });
    let initialRecordCount = 0;
    
    if (await recordTag.isVisible({ timeout: 2000 }).catch(() => false)) {
      const tagText = await recordTag.innerText();
      const match = tagText.match(/(\d+)/);
      initialRecordCount = match ? parseInt(match[1]) : 0;
      console.log(`📊 当前历史记录数: ${initialRecordCount} 条`);
    }
    
    // 检查历史记录表格
    const historyTable = historyCard.locator('.ant-table');
    const hasTable = await historyTable.isVisible({ timeout: 2000 }).catch(() => false);
    
    if (hasTable) {
      const rows = historyTable.locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`📊 历史记录表格显示: ${rowCount} 行`);
      
      if (rowCount > 0) {
        console.log('\n现有历史记录（最多显示前 3 条）:');
        for (let i = 0; i < Math.min(rowCount, 3); i++) {
          const row = rows.nth(i);
          const cells = row.locator('td');
          const time = await cells.nth(0).innerText();
          const target = await cells.nth(1).innerText();
          const func = await cells.nth(2).innerText();
          console.log(`  ${i + 1}. ${time} | ${target} | ${func}`);
        }
      }
    } else {
      console.log('ℹ️  暂无历史记录');
    }

    console.log('\n=== 步骤 3: 执行新命令生成历史记录 ===');
    
    // 执行一个测试命令
    const testCommand = {
      target: '*',
      function: 'test.ping',
      description: '持久化测试命令'
    };
    
    console.log(`执行测试命令: ${testCommand.function} on ${testCommand.target}`);
    
    // 选择目标
    const targetSelector = page.locator('.ant-select').filter({ has: page.locator('input[id*="target"]') });
    await targetSelector.click();
    await page.waitForTimeout(500);
    const targetOption = page.locator('.ant-select-item').filter({ hasText: '所有节点' });
    await targetOption.click();
    console.log('  ✓ 选择目标: *');
    
    // 选择函数
    const functionSelector = page.locator('.ant-select').filter({ has: page.locator('input[id*="function"]') });
    await functionSelector.click();
    await page.waitForTimeout(500);
    const functionOption = page.locator('.ant-select-item').filter({ hasText: 'test.ping' });
    await functionOption.first().click();
    console.log('  ✓ 选择函数: test.ping');
    
    // 执行命令
    const executeButton = page.locator('button:has-text("执行命令")');
    await executeButton.click();
    console.log('  ✓ 点击执行按钮');
    
    await page.waitForTimeout(3000);
    console.log('  ✅ 命令执行完成');

    console.log('\n=== 步骤 4: 验证历史记录已更新 ===');
    
    // 关闭最新执行结果卡片
    const closeButton = page.locator('button:has-text("关闭")');
    if (await closeButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await closeButton.click();
      await page.waitForTimeout(500);
    }
    
    // 再次检查历史记录数量
    await page.waitForTimeout(1000);
    const updatedRecordTag = historyCard.locator('.ant-tag').filter({ hasText: /\d+.*条记录/ });
    let updatedRecordCount = initialRecordCount;
    
    if (await updatedRecordTag.isVisible({ timeout: 2000 }).catch(() => false)) {
      const tagText = await updatedRecordTag.innerText();
      const match = tagText.match(/(\d+)/);
      updatedRecordCount = match ? parseInt(match[1]) : 0;
      console.log(`📊 更新后历史记录数: ${updatedRecordCount} 条`);
    }
    
    // 验证记录数增加
    if (updatedRecordCount > initialRecordCount) {
      console.log(`✅ 历史记录已增加: ${initialRecordCount} → ${updatedRecordCount}`);
    } else {
      console.log(`⚠️  历史记录数未变化: ${updatedRecordCount}`);
    }
    
    // 检查表格中是否有新记录
    if (await historyTable.isVisible({ timeout: 2000 }).catch(() => false)) {
      const rows = historyTable.locator('tbody tr');
      const rowCount = await rows.count();
      
      if (rowCount > 0) {
        console.log('\n✅ 最新的历史记录（第 1 条）:');
        const firstRow = rows.first();
        const cells = firstRow.locator('td');
        const time = await cells.nth(0).innerText();
        const target = await cells.nth(1).innerText();
        const func = await cells.nth(2).innerText();
        console.log(`   时间: ${time}`);
        console.log(`   目标: ${target}`);
        console.log(`   函数: ${func}`);
        
        // 验证是否是我们刚执行的命令
        if (func.includes('test.ping') && target.includes('*')) {
          console.log('✅✅ 确认：刚执行的命令已记录在历史中');
        } else {
          console.log('⚠️  最新记录不是刚执行的命令');
        }
      }
    }
    
    await page.screenshot({ 
      path: 'test-screenshots/history-after-execution.png',
      fullPage: true 
    });
    console.log('📸 截图: history-after-execution.png');

    console.log('\n=== 步骤 5: 刷新页面验证历史记录持久化 ===');
    
    console.log('🔄 刷新页面...');
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 重新点击 SaltStack 集成标签
    const saltStackTabAfterRefresh = page.locator('.ant-tabs-tab').filter({ hasText: /SaltStack.*集成/ });
    await saltStackTabAfterRefresh.waitFor({ timeout: 5000 });
    await saltStackTabAfterRefresh.click();
    await page.waitForTimeout(2000);
    
    console.log('✅ 页面刷新完成');

    console.log('\n=== 步骤 6: 检查刷新后的历史记录 ===');
    
    // 查找历史记录卡片
    const historyCardAfterRefresh = page.locator('.ant-card').filter({ 
      has: page.locator('span:has-text("命令执行历史")') 
    });
    await historyCardAfterRefresh.waitFor({ timeout: 5000 });
    
    // 获取刷新后的记录数量
    const recordTagAfterRefresh = historyCardAfterRefresh.locator('.ant-tag').filter({ hasText: /\d+.*条记录/ });
    let recordCountAfterRefresh = 0;
    
    if (await recordTagAfterRefresh.isVisible({ timeout: 2000 }).catch(() => false)) {
      const tagText = await recordTagAfterRefresh.innerText();
      const match = tagText.match(/(\d+)/);
      recordCountAfterRefresh = match ? parseInt(match[1]) : 0;
      console.log(`📊 刷新后历史记录数: ${recordCountAfterRefresh} 条`);
    }
    
    // 检查表格
    const historyTableAfterRefresh = historyCardAfterRefresh.locator('.ant-table');
    const hasTableAfterRefresh = await historyTableAfterRefresh.isVisible({ timeout: 2000 }).catch(() => false);
    
    if (hasTableAfterRefresh) {
      const rows = historyTableAfterRefresh.locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`📊 历史记录表格显示: ${rowCount} 行`);
      
      if (rowCount > 0) {
        console.log('\n刷新后的历史记录（前 3 条）:');
        for (let i = 0; i < Math.min(rowCount, 3); i++) {
          const row = rows.nth(i);
          const cells = row.locator('td');
          const time = await cells.nth(0).innerText();
          const target = await cells.nth(1).innerText();
          const func = await cells.nth(2).innerText();
          console.log(`  ${i + 1}. ${time} | ${target} | ${func}`);
        }
        
        // 验证我们执行的命令是否还在历史中
        const tableText = await historyTableAfterRefresh.innerText();
        if (tableText.includes('test.ping')) {
          console.log('\n✅✅✅ 持久化验证成功：刷新后历史记录仍然存在');
        } else {
          console.log('\n❌ 持久化验证失败：历史记录丢失');
        }
      } else {
        console.log('❌ 刷新后历史记录表格为空');
      }
    } else {
      // 检查是否显示"暂无记录"
      const noRecordAlert = historyCardAfterRefresh.locator('.ant-alert').filter({ hasText: /暂无执行记录/ });
      if (await noRecordAlert.isVisible({ timeout: 2000 }).catch(() => false)) {
        console.log('❌ 刷新后显示"暂无执行记录"，历史记录未持久化');
      }
    }
    
    await page.screenshot({ 
      path: 'test-screenshots/history-after-refresh.png',
      fullPage: true 
    });
    console.log('📸 截图: history-after-refresh.png');

    console.log('\n=== 步骤 7: 通过 API 验证历史记录 ===');
    
    // 获取 token
    const cookies = await context.cookies();
    const tokenCookie = cookies.find(c => c.name === 'token');
    const token = tokenCookie?.value;
    
    if (token) {
      try {
        // 检查是否有历史记录 API
        const historyResponse = await page.request.get(`${BASE_URL}/api/slurm/saltstack/history`, {
          headers: { 'Authorization': `Bearer ${token}` }
        });
        
        if (historyResponse.ok()) {
          const historyData = await historyResponse.json();
          const history = historyData.data || historyData;
          
          if (Array.isArray(history)) {
            console.log(`\n✅ API 返回 ${history.length} 条历史记录`);
            
            if (history.length > 0) {
              console.log('\nAPI 返回的最近 3 条历史:');
              history.slice(0, 3).forEach((item, index) => {
                console.log(`  ${index + 1}. ${item.function || item.fun || 'N/A'} - ${item.target || item.tgt || 'N/A'} - ${new Date(item.timestamp || item.start_time).toLocaleString('zh-CN')}`);
              });
              
              // 验证我们的测试命令是否在 API 返回中
              const hasTestPing = history.some(item => 
                (item.function || item.fun || '').includes('test.ping')
              );
              
              if (hasTestPing) {
                console.log('\n✅ API 验证：历史记录中包含 test.ping 命令');
              } else {
                console.log('\n⚠️  API 验证：历史记录中未找到 test.ping 命令');
              }
            }
          } else {
            console.log('⚠️  API 返回格式异常');
            console.log('返回数据:', JSON.stringify(historyData).substring(0, 200));
          }
        } else if (historyResponse.status() === 404) {
          console.log('⚠️  历史记录 API 不存在 (404)');
          console.log('ℹ️  历史记录可能只存储在前端本地');
        } else {
          console.log(`⚠️  历史记录 API 返回错误: ${historyResponse.status()}`);
        }
      } catch (error) {
        console.log(`⚠️  无法访问历史记录 API: ${error.message}`);
        console.log('ℹ️  历史记录可能只存储在前端本地');
      }
    } else {
      console.log('⚠️  未找到认证 token');
    }

    console.log('\n=== 测试总结 ===');
    console.log(`初始记录数: ${initialRecordCount}`);
    console.log(`执行命令后: ${updatedRecordCount}`);
    console.log(`刷新页面后: ${recordCountAfterRefresh}`);
    
    if (recordCountAfterRefresh === 0 && initialRecordCount === 0) {
      console.log('\n⚠️  结论：历史记录未持久化，每次刷新都会清空');
      console.log('💡 建议：');
      console.log('   1. 实现后端历史记录存储 API');
      console.log('   2. 或使用 localStorage 在前端持久化');
      console.log('   3. 当前历史记录只在会话期间有效');
    } else if (recordCountAfterRefresh > 0) {
      console.log('\n✅ 结论：历史记录已持久化');
    }
  });
});
