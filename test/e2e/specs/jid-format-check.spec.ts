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

test('检查 JID 格式差异', async ({ page }) => {
  console.log('\n========================================');
  console.log('JID 格式检查: 181709070486 vs 20251216150953871965');
  console.log('========================================\n');
  
  await login(page);
  
  // 1. 获取作业列表中的所有 JID
  console.log('1. 获取 /api/saltstack/jobs 返回的 JID 格式');
  const jobsResp = await page.evaluate(async (baseUrl: string) => {
    const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=50`, { credentials: 'include' });
    return await resp.json();
  }, BASE_URL);
  
  const jobs = jobsResp.data || [];
  console.log(`   返回作业数: ${jobs.length}`);
  
  if (jobs.length > 0) {
    console.log(`\n   JID 格式分析:`);
    const jidSet = new Set<string>();
    jobs.forEach((job: any) => {
      const jid = job.jid;
      const jidLen = jid?.length || 0;
      const jidFormat = jidLen >= 14 ? 'YYYYMMDDHHMMSSxxxxxx' : jidLen <= 12 ? '短格式' : '未知';
      jidSet.add(`len=${jidLen}`);
    });
    
    console.log(`   JID 长度分布: ${Array.from(jidSet).join(', ')}`);
    
    // 显示前 5 个 JID 详细信息
    console.log(`\n   前 5 个作业 JID:`);
    jobs.slice(0, 5).forEach((job: any, idx: number) => {
      const jid = job.jid;
      console.log(`     ${idx + 1}. JID="${jid}" (长度=${jid?.length})`);
      console.log(`        function=${job.function}, target=${job.target}`);
      console.log(`        task_id=${job.task_id || '无'}`);
      
      // 尝试解析 JID 为时间戳
      if (jid && jid.length >= 14) {
        const year = jid.substring(0, 4);
        const month = jid.substring(4, 6);
        const day = jid.substring(6, 8);
        const hour = jid.substring(8, 10);
        const min = jid.substring(10, 12);
        const sec = jid.substring(12, 14);
        console.log(`        解析时间: ${year}-${month}-${day} ${hour}:${min}:${sec}`);
      }
    });
  }
  
  // 2. 执行命令获取返回的 JID
  console.log(`\n2. 执行命令查看返回的 JID 格式`);
  const taskId = `JID-CHECK-${Date.now()}`;
  
  const execResp = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
    const resp = await fetch(`${params.baseUrl}/api/saltstack/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({
        target: '*',
        fun: 'cmd.run',
        arg: [`echo "jid-check-test"`],
        task_id: params.taskId
      })
    });
    return await resp.json();
  }, { baseUrl: BASE_URL, taskId });
  
  const execJid = execResp.jid;
  console.log(`   执行返回 JID: "${execJid}" (长度=${execJid?.length})`);
  console.log(`   执行返回 task_id: "${execResp.task_id}"`);
  
  if (execJid && execJid.length >= 14) {
    const year = execJid.substring(0, 4);
    const month = execJid.substring(4, 6);
    const day = execJid.substring(6, 8);
    const hour = execJid.substring(8, 10);
    const min = execJid.substring(10, 12);
    const sec = execJid.substring(12, 14);
    const micro = execJid.substring(14);
    console.log(`   JID 解析: ${year}-${month}-${day} ${hour}:${min}:${sec}.${micro}`);
  }
  
  // 3. 等待后检查作业是否出现在列表中
  console.log(`\n3. 等待 5 秒后检查作业列表`);
  await page.waitForTimeout(5000);
  
  const jobsAfter = await page.evaluate(async (baseUrl: string) => {
    const resp = await fetch(`${baseUrl}/api/saltstack/jobs?limit=50`, { credentials: 'include' });
    return await resp.json();
  }, BASE_URL);
  
  const jobsListAfter = jobsAfter.data || [];
  const foundNewJob = jobsListAfter.find((j: any) => j.jid === execJid);
  
  console.log(`   执行后作业数: ${jobsListAfter.length}`);
  console.log(`   找到新执行的作业: ${!!foundNewJob}`);
  
  if (foundNewJob) {
    console.log(`   新作业详情:`);
    console.log(`     JID: ${foundNewJob.jid}`);
    console.log(`     task_id: ${foundNewJob.task_id || '无'}`);
  } else {
    // 检查是否是 JID 格式问题
    console.log(`\n   !! 未在列表中找到新作业 !!`);
    console.log(`   可能原因:`);
    console.log(`     1. Salt 作业缓存延迟`);
    console.log(`     2. JID 格式不匹配`);
    
    // 比较 JID 格式
    if (jobsListAfter.length > 0) {
      const listJid = jobsListAfter[0].jid;
      console.log(`\n   JID 格式对比:`);
      console.log(`     列表 JID:   "${listJid}" (长度=${listJid?.length})`);
      console.log(`     执行 JID:   "${execJid}" (长度=${execJid?.length})`);
      console.log(`     格式一致:   ${listJid?.length === execJid?.length}`);
    }
  }
  
  // 4. 使用 by-task API 验证
  console.log(`\n4. 使用 by-task API 验证`);
  const byTaskResp = await page.evaluate(async (params: { baseUrl: string, taskId: string }) => {
    const resp = await fetch(`${params.baseUrl}/api/saltstack/jobs/by-task/${params.taskId}`, {
      credentials: 'include'
    });
    return {
      status: resp.status,
      data: await resp.json()
    };
  }, { baseUrl: BASE_URL, taskId });
  
  console.log(`   by-task API 状态: ${byTaskResp.status}`);
  if (byTaskResp.status === 200) {
    console.log(`   ✓ by-task API 成功找到任务`);
    console.log(`   返回的 JID: ${byTaskResp.data.jid}`);
  } else {
    console.log(`   ✗ by-task API 未找到任务: ${byTaskResp.data.error || '未知错误'}`);
  }
  
  console.log('\n========================================');
  console.log('测试完成');
  console.log('========================================');
});
