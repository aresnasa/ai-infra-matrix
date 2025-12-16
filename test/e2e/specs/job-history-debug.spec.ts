import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.48.123:8080';

/**
 * 登录为管理员
 */
async function login(page: any) {
  console.log('开始登录...');
  
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

test.describe('作业历史 TaskID 调试测试', () => {
  
  test('测试1: 直接调用 API 检查返回的 TaskID', async ({ page, request }) => {
    console.log('\n========================================');
    console.log('测试 SaltStack API 作业返回是否包含 task_id');
    console.log('========================================\n');
    
    // 先登录获取 Cookie
    await login(page);
    
    // 获取所有 cookies
    const cookies = await page.context().cookies();
    const cookieHeader = cookies.map(c => `${c.name}=${c.value}`).join('; ');
    
    // 测试1: 获取作业列表 API
    console.log('\n=== 调用 /api/saltstack/jobs API ===');
    const jobsResponse = await page.evaluate(async (baseUrl: string) => {
      const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=5`, {
        credentials: 'include'
      });
      return {
        status: resp.status,
        data: await resp.json()
      };
    }, BASE_URL);
    
    console.log('API 响应状态:', jobsResponse.status);
    console.log('返回数据:', JSON.stringify(jobsResponse.data, null, 2).substring(0, 2000));
    
    // 检查返回的 jobs 是否有 task_id 字段
    const jobs = jobsResponse.data?.data || [];
    console.log(`\n返回了 ${jobs.length} 个作业`);
    
    let jobsWithTaskId = 0;
    jobs.forEach((job: any, idx: number) => {
      console.log(`\n作业 ${idx + 1}:`);
      console.log(`  JID: ${job.jid}`);
      console.log(`  Function: ${job.function}`);
      console.log(`  task_id: ${job.task_id || '(无)'}`);
      console.log(`  TaskID: ${job.TaskID || '(无)'}`);
      console.log(`  taskId: ${job.taskId || '(无)'}`);
      
      if (job.task_id || job.TaskID || job.taskId) {
        jobsWithTaskId++;
      }
    });
    
    console.log(`\n=== 总结 ===`);
    console.log(`包含 TaskID 的作业数: ${jobsWithTaskId}/${jobs.length}`);
    
    // 先不断言，仅输出诊断信息
  });
  
  test('测试2: 执行命令并跟踪 TaskID 保存', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试执行命令后 TaskID 是否被正确保存到 Redis');
    console.log('========================================\n');
    
    await login(page);
    
    // 访问 SaltStack 页面
    console.log('\n访问 SaltStack 页面...');
    await page.goto(`${BASE_URL}/saltstack`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    await page.waitForTimeout(2000);
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/saltstack-page.png', 
      fullPage: true 
    });
    
    // 监听网络请求
    const apiCalls: any[] = [];
    page.on('request', (request: any) => {
      const url = request.url();
      if (url.includes('/api/saltstack')) {
        apiCalls.push({
          url,
          method: request.method(),
          postData: request.postData()
        });
      }
    });
    
    page.on('response', async (response: any) => {
      const url = response.url();
      if (url.includes('/api/saltstack/execute') || url.includes('/api/saltstack/jobs')) {
        console.log(`\n响应: ${response.request().method()} ${url}`);
        console.log(`状态: ${response.status()}`);
        try {
          const data = await response.json();
          console.log('响应数据:', JSON.stringify(data, null, 2).substring(0, 1500));
        } catch (e) {
          console.log('无法解析响应');
        }
      }
    });
    
    // 查找并点击"执行命令"或类似按钮，切换到执行Tab
    const executeTab = page.locator('.ant-tabs-tab').filter({ hasText: /执行|命令|Execute/ });
    const tabCount = await executeTab.count();
    console.log(`找到 ${tabCount} 个执行命令Tab`);
    
    if (tabCount > 0) {
      await executeTab.first().click();
      await page.waitForTimeout(1000);
    }
    
    // 截图当前状态
    await page.screenshot({ 
      path: 'test-results/saltstack-execute-tab.png', 
      fullPage: true 
    });
    
    // 查找命令输入框
    const cmdInput = page.locator('textarea, input').filter({ has: page.locator('[placeholder*="命令"], [placeholder*="command"]') });
    const inputCount = await cmdInput.count();
    console.log(`找到 ${inputCount} 个命令输入框`);
    
    // 尝试查找目标选择器
    const targetSelect = page.locator('.ant-select').first();
    const targetCount = await targetSelect.count();
    console.log(`找到 ${targetCount} 个目标选择器`);
    
    // 打印页面中的按钮
    const buttons = await page.locator('button').allTextContents();
    console.log('\n页面中的按钮:', buttons.filter(b => b.trim()).slice(0, 20));
    
    console.log('\n记录的 API 调用:', apiCalls.length);
    apiCalls.forEach((call, idx) => {
      console.log(`  ${idx + 1}. ${call.method} ${call.url}`);
    });
  });
  
  test('测试3: 检查作业历史页面是否正确显示 TaskID', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试作业历史页面显示');
    console.log('========================================\n');
    
    await login(page);
    
    // 访问 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    await page.waitForTimeout(2000);
    
    // 切换到作业历史 Tab
    const jobsTab = page.locator('.ant-tabs-tab').filter({ hasText: /作业|历史|Jobs|History/ });
    const jobsTabCount = await jobsTab.count();
    console.log(`找到 ${jobsTabCount} 个作业历史Tab`);
    
    if (jobsTabCount > 0) {
      await jobsTab.first().click();
      await page.waitForTimeout(2000);
    }
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/saltstack-jobs-tab.png', 
      fullPage: true 
    });
    
    // 查找作业列表表格
    const table = page.locator('.ant-table');
    const tableCount = await table.count();
    console.log(`找到 ${tableCount} 个表格`);
    
    if (tableCount > 0) {
      // 获取表头
      const headers = await table.locator('thead th').allTextContents();
      console.log('\n表头:', headers);
      
      // 检查是否有 TaskID 列
      const hasTaskIdColumn = headers.some(h => 
        h.includes('TaskID') || h.includes('任务ID') || h.includes('task_id')
      );
      console.log(`包含 TaskID 列: ${hasTaskIdColumn}`);
      
      // 获取前几行数据
      const rows = table.locator('tbody tr');
      const rowCount = await rows.count();
      console.log(`\n表格行数: ${rowCount}`);
      
      for (let i = 0; i < Math.min(rowCount, 5); i++) {
        const row = rows.nth(i);
        const cells = await row.locator('td').allTextContents();
        console.log(`\n行 ${i + 1}:`, cells);
      }
    }
    
    // 检查页面中是否有 "undefined" 显示
    const bodyText = await page.locator('body').textContent();
    const hasUndefined = bodyText?.includes('undefined');
    console.log(`\n页面包含 "undefined": ${hasUndefined}`);
    
    // 检查 localStorage 中的 JID-TaskID 映射
    const localStorageData = await page.evaluate(() => {
      const keys = Object.keys(localStorage).filter(k => 
        k.includes('jid') || k.includes('task') || k.includes('salt')
      );
      const result: Record<string, string> = {};
      keys.forEach(k => {
        result[k] = localStorage.getItem(k) || '';
      });
      return result;
    });
    
    console.log('\n=== localStorage 中的映射数据 ===');
    Object.entries(localStorageData).forEach(([key, value]) => {
      console.log(`  ${key}: ${String(value).substring(0, 200)}`);
    });
  });
  
  test('测试4: 完整执行流程并验证作业历史', async ({ page }) => {
    console.log('\n========================================');
    console.log('测试完整执行流程：执行命令 -> 检查作业历史');
    console.log('========================================\n');
    
    await login(page);
    
    // 访问 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    await page.waitForTimeout(2000);
    
    // 记录执行命令的 API 响应
    let executeResponse: any = null;
    let executeTaskId: string = '';
    
    page.on('response', async (response: any) => {
      const url = response.url();
      if (url.includes('/api/saltstack/execute')) {
        try {
          const data = await response.json();
          executeResponse = data;
          executeTaskId = data?.task_id || data?.data?.task_id || '';
          console.log('\n=== 执行命令 API 响应 ===');
          console.log(JSON.stringify(data, null, 2));
        } catch (e) {
          console.log('无法解析执行响应');
        }
      }
    });
    
    // 切换到执行命令 Tab
    const executeTab = page.locator('.ant-tabs-tab').filter({ hasText: /执行|批量|Execute|Batch/ });
    if (await executeTab.count() > 0) {
      await executeTab.first().click();
      await page.waitForTimeout(1000);
    }
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/before-execute.png', 
      fullPage: true 
    });
    
    // 尝试执行一个简单命令
    // 查找命令输入区域
    const textareas = page.locator('textarea');
    const textareaCount = await textareas.count();
    console.log(`找到 ${textareaCount} 个 textarea`);
    
    // 查找执行按钮
    const execButton = page.locator('button').filter({ hasText: /执行|Execute|运行|Run/ });
    const execButtonCount = await execButton.count();
    console.log(`找到 ${execButtonCount} 个执行按钮`);
    
    if (textareaCount > 0 && execButtonCount > 0) {
      // 输入测试命令
      await textareas.first().fill('echo "TaskID Test $(date)"');
      
      // 点击执行
      await execButton.first().click();
      
      console.log('\n等待执行完成...');
      await page.waitForTimeout(5000);
      
      // 截图执行后
      await page.screenshot({ 
        path: 'test-results/after-execute.png', 
        fullPage: true 
      });
      
      console.log('\n执行响应中的 TaskID:', executeTaskId);
    }
    
    // 等待更长时间让轮询完成
    await page.waitForTimeout(10000);
    
    // 切换到作业历史查看
    const jobsTab = page.locator('.ant-tabs-tab').filter({ hasText: /作业|历史|Jobs|History/ });
    if (await jobsTab.count() > 0) {
      await jobsTab.first().click();
      await page.waitForTimeout(2000);
    }
    
    // 截图作业历史
    await page.screenshot({ 
      path: 'test-results/job-history-after-execute.png', 
      fullPage: true 
    });
    
    // 获取最新的作业列表 API 响应
    const jobsApiResponse = await page.evaluate(async (baseUrl: string) => {
      const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=10`, {
        credentials: 'include'
      });
      return await resp.json();
    }, BASE_URL);
    
    console.log('\n=== 最新作业列表 API 响应 ===');
    const latestJobs = jobsApiResponse?.data || [];
    latestJobs.slice(0, 5).forEach((job: any, idx: number) => {
      console.log(`\n作业 ${idx + 1}:`);
      console.log(`  JID: ${job.jid}`);
      console.log(`  Function: ${job.function}`);
      console.log(`  task_id: ${job.task_id || '(无)'}`);
      console.log(`  Arguments: ${JSON.stringify(job.arguments)?.substring(0, 100)}`);
    });
    
    // 如果有 executeTaskId，使用新 API 查询
    if (executeTaskId) {
      console.log(`\n=== 使用 TaskID "${executeTaskId}" 查询作业 ===`);
      const taskJobResponse = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
        try {
          const resp = await fetch(`${params.baseUrl}/api/saltstack/jobs/by-task/${encodeURIComponent(params.taskId)}`, {
            credentials: 'include'
          });
          return {
            status: resp.status,
            data: await resp.json()
          };
        } catch (e) {
          return { status: 0, error: String(e) };
        }
      }, { baseUrl: BASE_URL, taskId: executeTaskId });
      
      console.log('查询结果:', JSON.stringify(taskJobResponse, null, 2));
    }
  });
});
