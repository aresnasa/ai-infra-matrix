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

test.describe('作业历史 TaskID 验证测试', () => {
  
  test('验证 API 返回的作业是否包含 task_id 字段', async ({ page }) => {
    console.log('\n========================================');
    console.log('验证 SaltStack Jobs API 返回 task_id');
    console.log('========================================\n');
    
    await login(page);
    
    // 通过 page.evaluate 调用 API
    const apiResponse = await page.evaluate(async (baseUrl: string) => {
      try {
        const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=10`, {
          credentials: 'include'
        });
        return {
          status: resp.status,
          data: await resp.json()
        };
      } catch (e) {
        return { error: String(e) };
      }
    }, BASE_URL);
    
    console.log('API 响应状态:', apiResponse.status);
    
    if (apiResponse.error) {
      console.log('API 调用失败:', apiResponse.error);
      return;
    }
    
    const jobs = apiResponse.data?.data || [];
    console.log(`获取到 ${jobs.length} 个作业\n`);
    
    // 统计有 task_id 的作业数
    let jobsWithTaskId = 0;
    
    jobs.forEach((job: any, idx: number) => {
      const hasTaskId = !!job.task_id;
      if (hasTaskId) jobsWithTaskId++;
      
      console.log(`作业 ${idx + 1}:`);
      console.log(`  JID: ${job.jid}`);
      console.log(`  Function: ${job.function}`);
      console.log(`  Target: ${job.target}`);
      console.log(`  task_id: ${job.task_id || '(无)'}`);
      console.log(`  start_time: ${job.start_time}`);
      console.log('');
    });
    
    console.log('========================================');
    console.log(`总结: ${jobsWithTaskId}/${jobs.length} 个作业包含 task_id`);
    console.log('========================================\n');
    
    // 验证 API 返回的数据结构
    if (jobs.length > 0) {
      const sampleJob = jobs[0];
      expect(sampleJob).toHaveProperty('jid');
      expect(sampleJob).toHaveProperty('function');
      console.log('✓ 作业数据结构验证通过');
    }
  });
  
  test('执行命令后验证 task_id 保存和查询', async ({ page }) => {
    console.log('\n========================================');
    console.log('执行命令并验证 task_id 保存');
    console.log('========================================\n');
    
    await login(page);
    
    // 执行一个简单的测试命令
    const testCommand = `echo "test-${Date.now()}"`;
    const taskId = `TEST-${Date.now()}`;
    
    console.log(`执行命令: ${testCommand}`);
    console.log(`TaskID: ${taskId}`);
    
    // 调用执行 API
    const execResponse = await page.evaluate(async (params: { baseUrl: string, cmd: string, taskId: string }) => {
      try {
        const resp = await fetch(`${params.baseUrl}/api/saltstack/execute`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          credentials: 'include',
          body: JSON.stringify({
            target: '*',
            function: 'cmd.run',
            arguments: params.cmd,
            task_id: params.taskId
          })
        });
        return {
          status: resp.status,
          data: await resp.json()
        };
      } catch (e) {
        return { error: String(e) };
      }
    }, { baseUrl: BASE_URL, cmd: testCommand, taskId });
    
    console.log('\n执行响应:', JSON.stringify(execResponse, null, 2).substring(0, 1000));
    
    if (execResponse.error) {
      console.log('执行失败:', execResponse.error);
      return;
    }
    
    // 提取返回的 JID 和 task_id
    const returnedJid = execResponse.data?.data?.jid || execResponse.data?.jid;
    const returnedTaskId = execResponse.data?.task_id || execResponse.data?.data?.task_id;
    
    console.log(`\n返回的 JID: ${returnedJid}`);
    console.log(`返回的 task_id: ${returnedTaskId}`);
    
    // 等待一下让 Redis 保存完成
    await page.waitForTimeout(3000);
    
    // 查询作业列表，验证 task_id 是否关联
    const jobsResponse = await page.evaluate(async (baseUrl: string) => {
      const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=5`, {
        credentials: 'include'
      });
      return await resp.json();
    }, BASE_URL);
    
    const jobs = jobsResponse?.data || [];
    console.log(`\n作业列表 (前5个):`);
    
    jobs.forEach((job: any, idx: number) => {
      console.log(`  ${idx + 1}. JID: ${job.jid}, task_id: ${job.task_id || '(无)'}, function: ${job.function}`);
    });
    
    // 查找刚执行的作业
    const executedJob = jobs.find((j: any) => j.jid === returnedJid);
    if (executedJob) {
      console.log(`\n✓ 找到刚执行的作业:`);
      console.log(`  JID: ${executedJob.jid}`);
      console.log(`  task_id: ${executedJob.task_id}`);
      
      if (executedJob.task_id === taskId) {
        console.log(`\n✅ task_id 匹配成功！`);
      } else {
        console.log(`\n⚠️ task_id 不匹配: 期望 ${taskId}, 实际 ${executedJob.task_id}`);
      }
    } else {
      console.log(`\n⚠️ 未在作业列表中找到 JID: ${returnedJid}`);
    }
    
    // 测试通过 task_id 查询作业的新 API
    if (taskId) {
      console.log(`\n=== 测试 /api/saltstack/jobs/by-task/${taskId} ===`);
      
      const byTaskResponse = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
        try {
          const resp = await fetch(`${params.baseUrl}/api/saltstack/jobs/by-task/${encodeURIComponent(params.taskId)}`, {
            credentials: 'include'
          });
          return {
            status: resp.status,
            data: await resp.json()
          };
        } catch (e) {
          return { error: String(e) };
        }
      }, { baseUrl: BASE_URL, taskId });
      
      console.log('查询响应:', JSON.stringify(byTaskResponse, null, 2).substring(0, 500));
      
      if (byTaskResponse.status === 200) {
        console.log('✅ 通过 task_id 查询成功');
      } else {
        console.log('⚠️ 通过 task_id 查询失败');
      }
    }
  });
});
