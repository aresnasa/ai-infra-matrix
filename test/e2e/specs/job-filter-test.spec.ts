import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.48.123:8080';

/**
 * 登录为管理员
 */
async function login(page: any) {
  console.log('开始登录到:', BASE_URL);
  
  await page.goto(`${BASE_URL}`, { 
    waitUntil: 'domcontentloaded',
    timeout: 60000 
  });
  
  await page.waitForTimeout(2000);
  
  const hasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  
  if (!hasLoginForm) {
    console.log('✓ 已自动登录');
    return;
  }
  
  console.log('填写登录凭据...');
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'admin123');
  await page.click('button[type="submit"]');
  
  await page.waitForTimeout(2000);
  console.log('✓ 登录完成');
}

test.describe('作业历史过滤功能测试', () => {
  
  test('测试作业历史页面过滤功能', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试作业历史过滤功能');
    console.log('========================================\n');
    
    await login(page);
    
    // 访问 SaltStack 页面
    console.log('访问 SaltStack 页面...');
    await page.goto(`${BASE_URL}/saltstack`, { 
      waitUntil: 'domcontentloaded',
      timeout: 60000 
    });
    
    await page.waitForTimeout(3000);
    
    // 截图初始状态
    await page.screenshot({ 
      path: 'test-results/saltstack-initial.png', 
      fullPage: true 
    });
    
    // 查找并点击作业历史 Tab
    console.log('\n=== 切换到作业历史 Tab ===');
    const allTabs = await page.locator('.ant-tabs-tab').allTextContents();
    console.log('所有 Tab:', allTabs);
    
    const jobsTab = page.locator('.ant-tabs-tab').filter({ hasText: /作业|历史|Jobs|History/ });
    const jobsTabCount = await jobsTab.count();
    console.log(`找到 ${jobsTabCount} 个作业历史 Tab`);
    
    if (jobsTabCount > 0) {
      await jobsTab.first().click();
      await page.waitForTimeout(2000);
      console.log('✓ 已切换到作业历史 Tab');
    }
    
    // 截图作业历史页面
    await page.screenshot({ 
      path: 'test-results/saltstack-jobs-tab.png', 
      fullPage: true 
    });
    
    // 查找搜索/过滤输入框
    console.log('\n=== 查找过滤控件 ===');
    
    // 查找所有输入框
    const inputs = page.locator('input');
    const inputCount = await inputs.count();
    console.log(`找到 ${inputCount} 个输入框`);
    
    for (let i = 0; i < inputCount; i++) {
      const input = inputs.nth(i);
      const placeholder = await input.getAttribute('placeholder');
      const id = await input.getAttribute('id');
      const name = await input.getAttribute('name');
      const type = await input.getAttribute('type');
      const visible = await input.isVisible();
      
      if (visible) {
        console.log(`  输入框 ${i + 1}: placeholder="${placeholder}", id="${id}", name="${name}", type="${type}"`);
      }
    }
    
    // 查找所有 Select 组件
    const selects = page.locator('.ant-select');
    const selectCount = await selects.count();
    console.log(`\n找到 ${selectCount} 个 Select 组件`);
    
    // 查找 TaskID 搜索框
    const taskIdInput = page.locator('input[placeholder*="TaskID"], input[placeholder*="任务ID"], input[placeholder*="task"]');
    const taskIdInputCount = await taskIdInput.count();
    console.log(`\n找到 ${taskIdInputCount} 个 TaskID 输入框`);
    
    // 查找通用搜索框
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="Search"], input[placeholder*="查询"]');
    const searchInputCount = await searchInput.count();
    console.log(`找到 ${searchInputCount} 个搜索输入框`);
    
    // 查找表格
    console.log('\n=== 检查作业列表表格 ===');
    const table = page.locator('.ant-table');
    const tableCount = await table.count();
    console.log(`找到 ${tableCount} 个表格`);
    
    if (tableCount > 0) {
      // 获取表头
      const headers = await table.locator('thead th').allTextContents();
      console.log('表头:', headers);
      
      // 检查是否有 TaskID 列
      const hasTaskIdColumn = headers.some(h => 
        h.includes('TaskID') || h.includes('任务ID') || h.includes('task')
      );
      console.log(`包含 TaskID 列: ${hasTaskIdColumn}`);
      
      // 获取前几行数据
      const rows = table.locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`\n表格行数: ${rowCount}`);
      
      if (rowCount > 0) {
        console.log('\n前3行数据:');
        for (let i = 0; i < Math.min(rowCount, 3); i++) {
          const row = rows.nth(i);
          const cells = await row.locator('td').allTextContents();
          console.log(`  行 ${i + 1}:`, cells.slice(0, 5).map(c => c.substring(0, 30)));
        }
      }
    }
    
    // 检查页面是否有 undefined 显示
    const bodyText = await page.locator('body').textContent();
    const hasUndefined = bodyText?.includes('undefined');
    console.log(`\n页面包含 "undefined": ${hasUndefined}`);
    
    // 检查是否有空状态
    const emptyState = page.locator('.ant-empty, .ant-table-empty');
    const emptyCount = await emptyState.count();
    console.log(`空状态组件数量: ${emptyCount}`);
  });
  
  test('测试 TaskID 过滤功能', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试 TaskID 过滤功能');
    console.log('========================================\n');
    
    await login(page);
    
    // 先执行一个命令获取 TaskID
    const taskId = `FILTER-TEST-${Date.now()}`;
    console.log(`执行命令并生成 TaskID: ${taskId}`);
    
    const execResponse = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
      const resp = await fetch(`${params.baseUrl}/api/saltstack/execute`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          target: '*',
          function: 'test.ping',
          task_id: params.taskId
        })
      });
      return {
        status: resp.status,
        data: await resp.json()
      };
    }, { baseUrl: BASE_URL, taskId });
    
    console.log('执行结果:', JSON.stringify(execResponse.data, null, 2).substring(0, 500));
    
    // 等待 Redis 保存
    await page.waitForTimeout(2000);
    
    // 访问 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`, { 
      waitUntil: 'domcontentloaded',
      timeout: 60000 
    });
    
    await page.waitForTimeout(3000);
    
    // 切换到作业历史 Tab
    const jobsTab = page.locator('.ant-tabs-tab').filter({ hasText: /作业|历史|Jobs|History/ });
    if (await jobsTab.count() > 0) {
      await jobsTab.first().click();
      await page.waitForTimeout(2000);
    }
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/before-filter.png', 
      fullPage: true 
    });
    
    // 尝试在 TaskID 输入框中输入
    console.log('\n=== 尝试过滤 TaskID ===');
    
    // 查找 TaskID 输入框
    const taskIdInputs = page.locator('input').filter({ 
      has: page.locator('[placeholder*="TaskID"], [placeholder*="任务"]') 
    });
    
    // 或者直接查找包含相关 placeholder 的输入框
    const allInputs = await page.locator('input').all();
    for (const input of allInputs) {
      const placeholder = await input.getAttribute('placeholder');
      if (placeholder && (placeholder.includes('TaskID') || placeholder.includes('任务') || placeholder.includes('搜索'))) {
        console.log(`找到输入框: ${placeholder}`);
        
        // 尝试输入 TaskID
        await input.fill(taskId);
        console.log(`已输入: ${taskId}`);
        
        // 等待过滤生效
        await page.waitForTimeout(1000);
        
        // 截图过滤后
        await page.screenshot({ 
          path: 'test-results/after-filter.png', 
          fullPage: true 
        });
        
        break;
      }
    }
    
    // 检查过滤结果
    const table = page.locator('.ant-table');
    if (await table.count() > 0) {
      const rows = table.locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`\n过滤后表格行数: ${rowCount}`);
      
      // 检查是否包含我们的 TaskID
      const tableText = await table.textContent();
      const hasOurTaskId = tableText?.includes(taskId);
      console.log(`表格包含 TaskID "${taskId}": ${hasOurTaskId}`);
    }
    
    // 使用 API 确认作业是否存在
    console.log('\n=== 通过 API 确认作业 ===');
    const apiResponse = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
      const resp = await fetch(`${params.baseUrl}/api/saltstack/jobs/by-task/${encodeURIComponent(params.taskId)}`, {
        credentials: 'include'
      });
      return {
        status: resp.status,
        data: await resp.json()
      };
    }, { baseUrl: BASE_URL, taskId });
    
    console.log('API 查询结果:', apiResponse.status);
    if (apiResponse.status === 200) {
      console.log('✓ 通过 API 能找到该任务');
    } else {
      console.log('✗ 通过 API 未能找到该任务');
    }
  });
});
