import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.48.123:8080';

async function login(page: any) {
  await page.goto(`${BASE_URL}`, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(2000);
  
  const hasLoginForm = await page.locator('input[type="text"], input[name="username"]').count() > 0;
  if (!hasLoginForm) return;
  
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
}

test('完整 API 调试', async ({ page }) => {
  console.log('\n========================================');
  console.log('完整 API 调试');
  console.log('========================================\n');
  
  await login(page);
  
  // 1. 获取当前作业列表
  console.log('1. 获取当前作业列表 (limit=50)');
  const jobsBefore = await page.evaluate(async (baseUrl: string) => {
    const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=50`, { credentials: 'include' });
    return await resp.json();
  }, BASE_URL);
  
  const jobsList = jobsBefore.data || [];
  console.log(`   返回作业数: ${jobsList.length}`);
  
  // 查看所有 JID
  console.log(`   JID 列表 (前10个):`);
  jobsList.slice(0, 10).forEach((job: any, idx: number) => {
    console.log(`     ${idx + 1}. JID=${job.jid}, task_id=${job.task_id || '无'}, func=${job.function}`);
  });
  
  // 2. 执行新命令
  const taskId = `API-DEBUG-${Date.now()}`;
  console.log(`\n2. 执行新命令, TaskID: ${taskId}`);
  
  const execResp = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
    const resp = await fetch(`${params.baseUrl}/api/saltstack/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({
        target: '*',
        fun: 'cmd.run',
        arg: [`echo "API-DEBUG-TEST"`],
        task_id: params.taskId
      })
    });
    return await resp.json();
  }, { baseUrl: BASE_URL, taskId });
  
  const newJid = execResp.jid;
  console.log(`   返回 JID: ${newJid}`);
  console.log(`   返回 task_id: ${execResp.task_id}`);
  
  // 等待
  console.log('\n   等待 5 秒让作业完成...');
  await page.waitForTimeout(5000);
  
  // 3. 再次获取作业列表
  console.log('\n3. 再次获取作业列表');
  const jobsAfter = await page.evaluate(async (baseUrl: string) => {
    const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=50`, { credentials: 'include' });
    return await resp.json();
  }, BASE_URL);
  
  const jobsListAfter = jobsAfter.data || [];
  console.log(`   返回作业数: ${jobsListAfter.length}`);
  
  // 查找新作业
  const foundNewJob = jobsListAfter.find((j: any) => j.jid === newJid);
  console.log(`   找到新作业 (JID=${newJid}): ${!!foundNewJob}`);
  
  if (foundNewJob) {
    console.log(`   新作业详情: ${JSON.stringify(foundNewJob)}`);
  } else {
    // 检查 JID 格式差异
    console.log(`\n   检查 JID 格式差异:`);
    console.log(`   - 新 JID: ${newJid} (长度=${newJid?.length})`);
    console.log(`   - 列表中 JID 示例: ${jobsListAfter[0]?.jid} (长度=${jobsListAfter[0]?.jid?.length})`);
    
    // 尝试模糊匹配
    const partialMatch = jobsListAfter.find((j: any) => 
      j.jid?.includes(newJid?.slice(-12)) || newJid?.includes(j.jid)
    );
    console.log(`   部分匹配: ${!!partialMatch}`);
    if (partialMatch) {
      console.log(`   匹配的 JID: ${partialMatch.jid}`);
    }
  }
  
  // 4. 测试 by-task API
  console.log('\n4. 测试 by-task API');
  const byTaskResp = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
    const resp = await fetch(`${params.baseUrl}/api/saltstack/jobs/by-task/${encodeURIComponent(params.taskId)}`, {
      credentials: 'include'
    });
    return { status: resp.status, data: await resp.json() };
  }, { baseUrl: BASE_URL, taskId });
  
  console.log(`   状态: ${byTaskResp.status}`);
  if (byTaskResp.status === 200) {
    console.log(`   JID: ${byTaskResp.data.jid}`);
    console.log(`   task_id: ${byTaskResp.data.task_id}`);
  }
  
  // 5. 检查有 task_id 的作业
  console.log('\n5. 检查有 task_id 的作业');
  const jobsWithTaskId = jobsListAfter.filter((j: any) => j.task_id);
  console.log(`   有 task_id 的作业数: ${jobsWithTaskId.length}/${jobsListAfter.length}`);
  
  // 总结
  console.log('\n========================================');
  console.log('问题诊断:');
  if (!foundNewJob) {
    console.log('❌ 新执行的作业未出现在 /api/saltstack/jobs 列表中');
    console.log('   可能原因:');
    console.log('   1. Salt /jobs API 返回的是缓存数据');
    console.log('   2. 作业尚未被 Salt 记录到历史中');
    console.log('   3. JID 格式不匹配');
  }
  if (jobsWithTaskId.length === 0) {
    console.log('❌ 作业列表中没有任何作业有 task_id');
    console.log('   可能原因:');
    console.log('   1. Redis 中的 JID-TaskID 映射的 JID 与列表中的 JID 不匹配');
    console.log('   2. 后端 GetSaltJobs 获取 TaskID 的逻辑有问题');
  }
  console.log('========================================\n');
});
