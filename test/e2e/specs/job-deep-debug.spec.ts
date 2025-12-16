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

test.describe('深度调试作业列表和过滤', () => {
  
  test('检查作业列表 API 返回和前端处理', async ({ page }) => {
    console.log('\n========================================');
    console.log('深度调试作业列表');
    console.log('========================================\n');
    
    await login(page);
    
    // 先执行一个命令
    const taskId = `DEBUG-${Date.now()}`;
    console.log(`1. 执行命令，TaskID: ${taskId}`);
    
    const execResp = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
      const resp = await fetch(`${params.baseUrl}/api/saltstack/execute`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          target: '*',
          fun: 'cmd.run',
          arg: [`echo "test-task-${params.taskId}"`],
          task_id: params.taskId
        })
      });
      return {
        status: resp.status,
        data: await resp.json()
      };
    }, { baseUrl: BASE_URL, taskId });
    
    const returnedJid = execResp.data?.jid;
    console.log(`   执行状态: ${execResp.status}`);
    console.log(`   返回 JID: ${returnedJid}`);
    console.log(`   返回 task_id: ${execResp.data?.task_id}`);
    
    // 等待 Redis 保存
    await page.waitForTimeout(3000);
    
    // 2. 直接调用 API 获取作业列表
    console.log(`\n2. 调用 /api/saltstack/jobs API`);
    const jobsResp = await page.evaluate(async (baseUrl: string) => {
      const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=20`, {
        credentials: 'include'
      });
      return {
        status: resp.status,
        data: await resp.json()
      };
    }, BASE_URL);
    
    console.log(`   API 状态: ${jobsResp.status}`);
    const jobs = jobsResp.data?.data || [];
    console.log(`   返回作业数: ${jobs.length}`);
    
    // 打印前 3 个作业的完整结构
    console.log(`   前3个作业结构:`);
    jobs.slice(0, 3).forEach((j: any, idx: number) => {
      console.log(`   [${idx}] ${JSON.stringify(j).substring(0, 200)}`);
    });
    
    // 检查是否有新执行的作业
    const newJob = jobs.find((j: any) => j.jid === returnedJid);
    console.log(`   找到新作业 (JID=${returnedJid}): ${!!newJob}`);
    
    if (newJob) {
      console.log(`   新作业详情:`);
      console.log(`     - jid: ${newJob.jid}`);
      console.log(`     - task_id: ${newJob.task_id}`);
      console.log(`     - function: ${newJob.function}`);
      console.log(`     - target: ${newJob.target}`);
    }
    
    // 检查所有有 task_id 的作业
    const jobsWithTaskId = jobs.filter((j: any) => j.task_id);
    console.log(`\n   带 task_id 的作业: ${jobsWithTaskId.length}/${jobs.length}`);
    
    if (jobsWithTaskId.length > 0) {
      console.log(`   示例:`);
      jobsWithTaskId.slice(0, 3).forEach((j: any) => {
        console.log(`     - JID: ${j.jid}, task_id: ${j.task_id}`);
      });
    }
    
    // 3. 访问前端页面，查看 jobs 状态
    console.log(`\n3. 访问前端页面检查`);
    await page.goto(`${BASE_URL}/saltstack`, { 
      waitUntil: 'domcontentloaded',
      timeout: 60000 
    });
    
    await page.waitForTimeout(3000);
    
    // 切换到作业历史 Tab
    const jobsTab = page.locator('.ant-tabs-tab').filter({ hasText: /作业.*历史|Jobs/ });
    if (await jobsTab.count() > 0) {
      await jobsTab.first().click();
      await page.waitForTimeout(2000);
    }
    
    // 通过 JavaScript 获取 React 组件中的 jobs 状态
    console.log(`\n4. 检查前端组件中的 jobs 数据`);
    const frontendJobs = await page.evaluate(() => {
      // 尝试从 window 获取 React 数据
      // 通过 DOM 元素检查
      const tableRows = document.querySelectorAll('.ant-table-tbody tr');
      const rowsData: any[] = [];
      
      tableRows.forEach((row, idx) => {
        const cells = row.querySelectorAll('td');
        if (cells.length > 0) {
          rowsData.push({
            index: idx,
            cellCount: cells.length,
            firstCell: cells[0]?.textContent?.substring(0, 50),
            secondCell: cells[1]?.textContent?.substring(0, 50),
          });
        }
      });
      
      return {
        rowCount: tableRows.length,
        rows: rowsData.slice(0, 5)
      };
    });
    
    console.log(`   表格行数: ${frontendJobs.rowCount}`);
    console.log(`   前5行数据:`);
    frontendJobs.rows.forEach((row: any) => {
      console.log(`     ${row.index}: ${row.firstCell} | ${row.secondCell}`);
    });
    
    // 5. 尝试搜索 TaskID
    console.log(`\n5. 尝试搜索 TaskID: ${taskId}`);
    
    // 查找搜索框
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="任务"]').first();
    if (await searchInput.count() > 0) {
      await searchInput.fill(taskId);
      await page.waitForTimeout(1500);
      
      // 检查过滤后的表格
      const filteredRows = await page.evaluate(() => {
        const rows = document.querySelectorAll('.ant-table-tbody tr');
        return {
          count: rows.length,
          isEmpty: document.querySelector('.ant-empty') !== null
        };
      });
      
      console.log(`   过滤后行数: ${filteredRows.count}`);
      console.log(`   是否显示空状态: ${filteredRows.isEmpty}`);
    }
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/debug-jobs-filtered.png', 
      fullPage: true 
    });
    
    // 6. 通过 by-task API 确认
    console.log(`\n6. 通过 by-task API 确认`);
    const byTaskResp = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
      const resp = await fetch(`${params.baseUrl}/api/saltstack/jobs/by-task/${encodeURIComponent(params.taskId)}`, {
        credentials: 'include'
      });
      return {
        status: resp.status,
        data: await resp.json()
      };
    }, { baseUrl: BASE_URL, taskId });
    
    console.log(`   API 状态: ${byTaskResp.status}`);
    if (byTaskResp.status === 200) {
      console.log(`   ✓ 找到任务`);
      console.log(`   JID: ${byTaskResp.data?.jid}`);
    } else {
      console.log(`   ✗ 未找到任务`);
      console.log(`   错误: ${JSON.stringify(byTaskResp.data)}`);
    }
    
    // 总结
    console.log('\n========================================');
    console.log('诊断总结:');
    console.log(`  - 执行任务: ${execResp.status === 200 ? '✓' : '✗'}`);
    console.log(`  - API 返回 task_id: ${execResp.data?.task_id ? '✓' : '✗'}`);
    console.log(`  - 作业列表包含新任务: ${!!newJob ? '✓' : '✗'}`);
    console.log(`  - 作业有 task_id 字段: ${newJob?.task_id ? '✓' : '✗'}`);
    console.log(`  - by-task API 可查: ${byTaskResp.status === 200 ? '✓' : '✗'}`);
    console.log('========================================\n');
  });
});
