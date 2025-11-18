// @ts-nocheck
/* eslint-disable */
/**
 * SLURM前端异步加载和扩容功能测试
 * 
 * 测试内容:
 * 1. 验证页面框架立即显示（无全屏加载阻塞）
 * 2. 验证骨架屏显示
 * 3. 验证数据异步分阶段加载
 * 4. 测试SLURM集群扩容功能
 * 5. 验证集群状态获取
 */

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://192.168.0.200:8080';

/**
 * 登录辅助函数
 */
async function loginIfNeeded(page) {
  await page.goto(BASE + '/');
  await page.waitForLoadState('domcontentloaded');
  
  // 检查是否需要登录
  const needsLogin = await page.getByRole('tab', { name: '登录' }).isVisible({ timeout: 5000 }).catch(() => false);
  
  if (needsLogin) {
    const user = process.env.E2E_USER || 'admin';
    const pass = process.env.E2E_PASS || 'admin123';
    
    await page.getByPlaceholder('用户名').fill(user);
    await page.getByPlaceholder('密码').fill(pass);
    await page.getByRole('button', { name: '登录' }).click();
    
    // 等待登录成功
    await expect(page).toHaveURL(/\/(projects|)$/, { timeout: 10000 });
  }
}

test.describe('SLURM前端异步加载测试', () => {
  
  test('应该立即显示页面框架，无全屏加载阻塞', async ({ page }) => {
    await loginIfNeeded(page);
    
    // 记录页面加载开始时间
    const startTime = Date.now();
    
    // 导航到SLURM页面
    await page.goto(BASE + '/slurm');
    
    // 检查页面标题是否立即显示（不等待数据加载）
    const titleVisible = await page.getByText('SLURM 集群管理').isVisible({ timeout: 500 });
    const loadTime = Date.now() - startTime;
    
    console.log(`✓ 页面框架显示时间: ${loadTime}ms`);
    expect(titleVisible).toBe(true);
    expect(loadTime).toBeLessThan(1000); // 页面框架应该在1秒内显示
    
    // 验证关键按钮立即可见
    await expect(page.getByRole('button', { name: '刷新' })).toBeVisible({ timeout: 500 });
    await expect(page.getByRole('button', { name: '扩容节点' })).toBeVisible({ timeout: 500 });
    
    console.log('✓ 页面框架和按钮立即显示，无全屏加载阻塞');
  });

  test('应该显示骨架屏加载状态', async ({ page }) => {
    await loginIfNeeded(page);
    await page.goto(BASE + '/slurm');
    
    // 等待页面框架加载
    await page.waitForLoadState('domcontentloaded');
    
    // 检查是否有骨架屏元素（Ant Design Skeleton组件特征）
    // 骨架屏会有 ant-skeleton 类名
    const skeletonElements = page.locator('.ant-skeleton');
    const hasSkeletons = await skeletonElements.count() > 0;
    
    if (hasSkeletons) {
      console.log('✓ 检测到骨架屏加载状态');
      
      // 等待骨架屏消失（数据加载完成）
      await page.waitForSelector('.ant-skeleton', { state: 'hidden', timeout: 10000 });
      console.log('✓ 骨架屏已消失，数据加载完成');
    } else {
      console.log('ℹ️  数据加载速度较快，未捕获到骨架屏状态');
    }
  });

  test('应该异步分阶段加载数据', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    // 监听API请求顺序
    const apiCalls = [];
    
    page.on('request', req => {
      const url = req.url();
      if (url.includes('/api/slurm/')) {
        const timestamp = Date.now();
        const endpoint = url.split('/api/slurm/')[1].split('?')[0];
        apiCalls.push({ endpoint, timestamp, url });
        console.log(`📡 API调用: ${endpoint} at ${timestamp}`);
      }
    });
    
    const startTime = Date.now();
    await page.goto(BASE + '/slurm');
    
    // 等待页面加载完成
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    
    console.log('\n📊 API调用顺序分析:');
    console.log('─'.repeat(80));
    
    if (apiCalls.length > 0) {
      const firstCall = apiCalls[0].timestamp;
      
      apiCalls.forEach(call => {
        const delay = call.timestamp - firstCall;
        console.log(`${delay.toString().padStart(6)}ms | ${call.endpoint}`);
      });
      
      console.log('─'.repeat(80));
      
      // 验证核心数据（summary/nodes）应该最先加载
      const summaryCall = apiCalls.find(c => c.endpoint.includes('summary'));
      const nodesCall = apiCalls.find(c => c.endpoint.includes('nodes'));
      
      if (summaryCall && nodesCall) {
        const summaryDelay = summaryCall.timestamp - firstCall;
        const nodesDelay = nodesCall.timestamp - firstCall;
        
        console.log(`\n✓ Summary加载延迟: ${summaryDelay}ms`);
        console.log(`✓ Nodes加载延迟: ${nodesDelay}ms`);
        
        // 核心数据应该在500ms内开始加载
        expect(summaryDelay).toBeLessThan(500);
        expect(nodesDelay).toBeLessThan(500);
      }
    }
  });
});

test.describe('SLURM集群状态测试', () => {
  
  test('应该正确获取并显示集群摘要信息', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    // 先通过API获取数据
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    // 获取集群摘要
    const summaryResponse = await request.get(BASE + '/api/slurm/summary', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    expect(summaryResponse.ok()).toBeTruthy();
    const summaryData = await summaryResponse.json();
    
    console.log('\n📊 集群摘要数据:');
    console.log(JSON.stringify(summaryData, null, 2));
    
    // 访问页面验证显示
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 截图保存当前状态
    await page.screenshot({ 
      path: 'test-screenshots/slurm-cluster-summary.png',
      fullPage: true 
    });
    
    console.log('✓ 集群摘要页面截图已保存');
    
    // 验证关键统计信息可见
    const summaryCards = page.locator('.ant-statistic');
    const cardCount = await summaryCards.count();
    
    console.log(`✓ 找到 ${cardCount} 个统计卡片`);
    expect(cardCount).toBeGreaterThan(0);
  });

  test('应该正确获取并显示节点列表', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    // 通过API获取节点列表
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    const nodesResponse = await request.get(BASE + '/api/slurm/nodes', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    expect(nodesResponse.ok()).toBeTruthy();
    const nodesData = await nodesResponse.json();
    
    console.log('\n📋 节点列表数据:');
    console.log(JSON.stringify(nodesData, null, 2));
    
    if (nodesData.data && Array.isArray(nodesData.data)) {
      console.log(`\n总节点数: ${nodesData.data.length}`);
      console.log(`Demo模式: ${nodesData.demo ? '是' : '否'}`);
      
      // 显示节点详情
      nodesData.data.forEach((node, index) => {
        console.log(`\n节点 ${index + 1}:`);
        console.log(`  名称: ${node.name}`);
        console.log(`  状态: ${node.state}`);
        console.log(`  CPU: ${node.cpus}`);
        console.log(`  内存: ${node.memory_mb}MB`);
        console.log(`  分区: ${node.partition}`);
      });
    }
    
    // 访问页面验证显示
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 点击节点管理标签
    await page.getByText('节点管理').click();
    await page.waitForTimeout(1000);
    
    // 截图保存
    await page.screenshot({ 
      path: 'test-screenshots/slurm-nodes-list.png',
      fullPage: true 
    });
    
    console.log('\n✓ 节点列表页面截图已保存');
  });

  test('应该正确获取作业队列信息', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    const jobsResponse = await request.get(BASE + '/api/slurm/jobs', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    expect(jobsResponse.ok()).toBeTruthy();
    const jobsData = await jobsResponse.json();
    
    console.log('\n📋 作业队列数据:');
    console.log(JSON.stringify(jobsData, null, 2));
    
    // 访问页面
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 点击作业队列标签
    await page.getByText('作业队列').click();
    await page.waitForTimeout(1000);
    
    await page.screenshot({ 
      path: 'test-screenshots/slurm-jobs-queue.png',
      fullPage: true 
    });
    
    console.log('✓ 作业队列页面截图已保存');
  });
});

test.describe('SLURM集群扩容功能测试', () => {
  
  test('应该能够打开扩容节点对话框', async ({ page }) => {
    await loginIfNeeded(page);
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 点击扩容节点按钮
    const scaleUpButton = page.getByRole('button', { name: '扩容节点' });
    await expect(scaleUpButton).toBeVisible();
    await scaleUpButton.click();
    
    // 等待对话框出现
    await page.waitForSelector('.ant-modal', { timeout: 5000 });
    
    // 验证对话框标题
    const modalTitle = await page.locator('.ant-modal-title').textContent();
    console.log(`✓ 对话框标题: ${modalTitle}`);
    
    // 截图保存
    await page.screenshot({ 
      path: 'test-screenshots/slurm-scale-up-modal.png',
      fullPage: true 
    });
    
    console.log('✓ 扩容对话框已打开');
    
    // 关闭对话框
    await page.getByRole('button', { name: '取消' }).click();
  });

  test('应该验证节点模板功能', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    // 获取节点模板
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    const templatesResponse = await request.get(BASE + '/api/slurm/node-templates', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }).catch(() => null);
    
    if (templatesResponse && templatesResponse.ok()) {
      const templatesData = await templatesResponse.json();
      console.log('\n📋 节点模板:');
      console.log(JSON.stringify(templatesData, null, 2));
    } else {
      console.log('ℹ️  节点模板API不可用或返回错误');
    }
    
    // 访问页面
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 点击管理模板按钮
    const templateButton = page.getByRole('button', { name: '管理模板' }).first();
    if (await templateButton.isVisible()) {
      await templateButton.click();
      await page.waitForTimeout(1000);
      
      await page.screenshot({ 
        path: 'test-screenshots/slurm-node-templates.png',
        fullPage: true 
      });
      
      console.log('✓ 节点模板对话框截图已保存');
      
      // 关闭对话框
      const cancelButton = page.getByRole('button', { name: '取消' });
      if (await cancelButton.isVisible()) {
        await cancelButton.click();
      }
    }
  });

  test('应该测试SLURM命令执行功能', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    // 测试sinfo命令
    console.log('\n🔧 测试SLURM命令执行...');
    
    const execResponse = await request.post(BASE + '/api/slurm/exec', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: {
        command: 'sinfo'
      }
    });
    
    expect(execResponse.ok()).toBeTruthy();
    const execData = await execResponse.json();
    
    console.log('📡 sinfo 命令结果:');
    console.log(execData.output || execData.stdout);
    
    expect(execData.success).toBe(true);
    
    // 测试sinfo -Nel命令
    const detailResponse = await request.post(BASE + '/api/slurm/exec', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: {
        command: 'sinfo -Nel'
      }
    });
    
    if (detailResponse.ok()) {
      const detailData = await detailResponse.json();
      console.log('\n📡 sinfo -Nel 详细输出:');
      console.log(detailData.output || detailData.stdout);
    }
  });

  test('应该获取SLURM诊断信息', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    // 获取诊断信息
    console.log('\n🔍 获取SLURM诊断信息...');
    
    const diagResponse = await request.get(BASE + '/api/slurm/diagnostics', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    expect(diagResponse.ok()).toBeTruthy();
    const diagData = await diagResponse.json();
    
    console.log('\n📊 诊断信息:');
    console.log('─'.repeat(80));
    
    if (diagData.diagnostics) {
      if (diagData.diagnostics.sinfo) {
        console.log('\n1️⃣  sinfo 输出:');
        console.log(diagData.diagnostics.sinfo);
      }
      
      if (diagData.diagnostics.sinfo_detail) {
        console.log('\n2️⃣  sinfo -Nel 详细输出:');
        console.log(diagData.diagnostics.sinfo_detail);
      }
      
      if (diagData.diagnostics.squeue) {
        console.log('\n3️⃣  squeue 输出:');
        console.log(diagData.diagnostics.squeue);
      }
    }
    
    console.log('─'.repeat(80));
    console.log(`✓ 诊断信息获取时间: ${diagData.timestamp}`);
  });
});

test.describe('SLURM性能和用户体验测试', () => {
  
  test('应该测试页面完整加载时间', async ({ page }) => {
    await loginIfNeeded(page);
    
    const startTime = Date.now();
    
    await page.goto(BASE + '/slurm');
    
    // 等待页面框架显示
    await page.waitForSelector('h2', { timeout: 2000 });
    const frameTime = Date.now() - startTime;
    
    // 等待网络空闲
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    const totalTime = Date.now() - startTime;
    
    console.log('\n⏱️  性能指标:');
    console.log('─'.repeat(80));
    console.log(`  页面框架显示时间: ${frameTime}ms`);
    console.log(`  完整加载时间: ${totalTime}ms`);
    console.log(`  性能提升: ${frameTime < 1000 ? '✅ 优秀' : '⚠️  需优化'}`);
    console.log('─'.repeat(80));
    
    expect(frameTime).toBeLessThan(2000); // 页面框架应该在2秒内显示
    expect(totalTime).toBeLessThan(10000); // 完整加载应该在10秒内完成
  });

  test('应该测试刷新功能', async ({ page }) => {
    await loginIfNeeded(page);
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 点击刷新按钮
    const refreshButton = page.getByRole('button', { name: '刷新' });
    await expect(refreshButton).toBeVisible();
    
    console.log('\n🔄 测试刷新功能...');
    const startTime = Date.now();
    
    await refreshButton.click();
    
    // 等待刷新完成（观察loading状态）
    await page.waitForTimeout(2000);
    
    const refreshTime = Date.now() - startTime;
    console.log(`✓ 刷新完成时间: ${refreshTime}ms`);
    
    await page.screenshot({ 
      path: 'test-screenshots/slurm-after-refresh.png',
      fullPage: true 
    });
  });
});
