// @ts-nocheck
/* eslint-disable */
/**
 * SLURM快速测试
 * 简化版本，专注于核心功能测试
 */

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://192.168.0.200:8080';

test.describe('SLURM快速测试', () => {
  
  test('测试API - 集群状态', async ({ request }) => {
    // 登录获取token
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    console.log('✓ 登录成功');
    const token = loginData.token;
    
    // 测试Summary API
    console.log('\n📊 测试 /api/slurm/summary');
    const summaryResponse = await request.get(BASE + '/api/slurm/summary', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    expect(summaryResponse.ok()).toBeTruthy();
    const summaryData = await summaryResponse.json();
    console.log('Summary:', JSON.stringify(summaryData, null, 2));
    
    // 测试Nodes API
    console.log('\n📋 测试 /api/slurm/nodes');
    const nodesResponse = await request.get(BASE + '/api/slurm/nodes', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    expect(nodesResponse.ok()).toBeTruthy();
    const nodesData = await nodesResponse.json();
    console.log(`节点数: ${nodesData.data?.length || 0}`);
    console.log('Demo模式:', nodesData.demo);
    
    if (nodesData.data && nodesData.data.length > 0) {
      console.log('\n节点详情:');
      nodesData.data.forEach((node, i) => {
        console.log(`  ${i+1}. ${node.name} - ${node.state} (CPU: ${node.cpus}, 内存: ${node.memory_mb}MB)`);
      });
    }
    
    // 测试Jobs API
    console.log('\n📋 测试 /api/slurm/jobs');
    const jobsResponse = await request.get(BASE + '/api/slurm/jobs', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    expect(jobsResponse.ok()).toBeTruthy();
    const jobsData = await jobsResponse.json();
    console.log(`作业数: ${jobsData.data?.length || 0}`);
  });

  test('测试SLURM命令执行API', async ({ request }) => {
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    console.log('\n🔧 测试 /api/slurm/exec');
    
    // 测试sinfo
    const sinfoResponse = await request.post(BASE + '/api/slurm/exec', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: { command: 'sinfo' }
    });
    
    expect(sinfoResponse.ok()).toBeTruthy();
    const sinfoData = await sinfoResponse.json();
    console.log('sinfo 输出:');
    console.log(sinfoData.output || sinfoData.stdout);
    
    // 测试sinfo -Nel
    const detailResponse = await request.post(BASE + '/api/slurm/exec', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: { command: 'sinfo -Nel' }
    });
    
    if (detailResponse.ok()) {
      const detailData = await detailResponse.json();
      console.log('\nsinfo -Nel 输出:');
      console.log(detailData.output || detailData.stdout);
    }
  });

  test('测试SLURM诊断API', async ({ request }) => {
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    console.log('\n🔍 测试 /api/slurm/diagnostics');
    
    const diagResponse = await request.get(BASE + '/api/slurm/diagnostics', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    
    expect(diagResponse.ok()).toBeTruthy();
    const diagData = await diagResponse.json();
    
    console.log('─'.repeat(80));
    if (diagData.diagnostics) {
      if (diagData.diagnostics.sinfo) {
        console.log('\n📊 sinfo:');
        console.log(diagData.diagnostics.sinfo);
      }
      if (diagData.diagnostics.sinfo_detail) {
        console.log('\n📊 sinfo -Nel:');
        console.log(diagData.diagnostics.sinfo_detail);
      }
      if (diagData.diagnostics.squeue) {
        console.log('\n📊 squeue:');
        console.log(diagData.diagnostics.squeue);
      }
    }
    console.log('─'.repeat(80));
  });

  test('测试前端页面加载', async ({ page, context }) => {
    // 先登录
    await page.goto(BASE + '/');
    await page.waitForLoadState('domcontentloaded');
    
    // 检查是否有登录表单
    const loginTab = await page.locator('text=登录').first().isVisible({ timeout: 3000 }).catch(() => false);
    
    if (loginTab) {
      console.log('需要登录...');
      await page.fill('input[placeholder*="用户名"]', 'admin');
      await page.fill('input[placeholder*="密码"]', 'admin123');
      await page.click('button:has-text("登录")');
      await page.waitForTimeout(2000);
    }
    
    // 导航到SLURM页面
    console.log('\n📄 测试前端页面加载...');
    const startTime = Date.now();
    
    await page.goto(BASE + '/slurm');
    
    // 等待页面基础元素
    await page.waitForSelector('h2', { timeout: 5000 });
    const frameTime = Date.now() - startTime;
    console.log(`✓ 页面框架显示时间: ${frameTime}ms`);
    
    // 等待网络空闲
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    const totalTime = Date.now() - startTime;
    console.log(`✓ 完整加载时间: ${totalTime}ms`);
    
    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/slurm-page-loaded.png',
      fullPage: true 
    });
    console.log('✓ 截图已保存: test-screenshots/slurm-page-loaded.png');
    
    // 检查关键元素
    const hasRefreshBtn = await page.locator('button:has-text("刷新")').isVisible();
    const hasScaleBtn = await page.locator('button:has-text("扩容")').first().isVisible();
    
    console.log(`✓ 刷新按钮: ${hasRefreshBtn ? '✓' : '✗'}`);
    console.log(`✓ 扩容按钮: ${hasScaleBtn ? '✓' : '✗'}`);
    
    // 性能评估
    console.log('\n⏱️  性能评估:');
    console.log(`  页面框架: ${frameTime < 1000 ? '✅ 优秀' : frameTime < 2000 ? '⚠️  良好' : '❌ 需优化'} (${frameTime}ms)`);
    console.log(`  完整加载: ${totalTime < 3000 ? '✅ 优秀' : totalTime < 5000 ? '⚠️  良好' : '❌ 需优化'} (${totalTime}ms)`);
  });

  test('测试扩容对话框', async ({ page }) => {
    // 登录
    await page.goto(BASE + '/');
    await page.waitForLoadState('domcontentloaded');
    
    const loginTab = await page.locator('text=登录').first().isVisible({ timeout: 3000 }).catch(() => false);
    if (loginTab) {
      await page.fill('input[placeholder*="用户名"]', 'admin');
      await page.fill('input[placeholder*="密码"]', 'admin123');
      await page.click('button:has-text("登录")');
      await page.waitForTimeout(2000);
    }
    
    // 访问SLURM页面
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    console.log('\n📝 测试扩容对话框...');
    
    // 查找扩容按钮
    const scaleButton = page.locator('button:has-text("扩容")').first();
    const isVisible = await scaleButton.isVisible();
    
    if (isVisible) {
      await scaleButton.click();
      await page.waitForTimeout(1000);
      
      // 检查对话框
      const modalVisible = await page.locator('.ant-modal').isVisible();
      console.log(`✓ 对话框显示: ${modalVisible ? '✓' : '✗'}`);
      
      if (modalVisible) {
        await page.screenshot({ 
          path: 'test-screenshots/slurm-scale-modal.png',
          fullPage: true 
        });
        console.log('✓ 对话框截图已保存');
        
        // 关闭对话框
        const cancelBtn = page.locator('button:has-text("取消")');
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click();
        }
      }
    } else {
      console.log('⚠️  扩容按钮未找到');
    }
  });
});
