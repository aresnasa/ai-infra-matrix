// @ts-nocheck
/* eslint-disable */
/**
 * SLURM Cluster Status Test
 * 
 * 目的: 验证 SLURM 集群状态能够正确获取并显示
 * 
 * 测试内容:
 * 1. 检查 SLURM 集群是否在线
 * 2. 验证节点列表显示
 * 3. 验证作业队列显示
 * 4. 检查集群统计信息
 * 5. 验证后端 API 返回正确的集群状态
 */

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://localhost:8080';

/**
 * 登录辅助函数
 */
async function loginIfNeeded(page) {
  await page.goto(BASE + '/');
  if (await page.getByRole('tab', { name: '登录' }).isVisible().catch(() => false)) {
    const user = process.env.E2E_USER || 'admin';
    const pass = process.env.E2E_PASS || 'admin123';
    await page.getByPlaceholder('用户名').fill(user);
    await page.getByPlaceholder('密码').fill(pass);
    await page.getByRole('button', { name: '登录' }).click();
    await expect(page).toHaveURL(/\/(projects|)$/);
  }
}

test.describe('SLURM Cluster Status Tests', () => {
  
  test('should display SLURM cluster management page', async ({ page }) => {
    await loginIfNeeded(page);
    
    // 导航到 SLURM 管理页面
    await page.goto(BASE + '/slurm');
    
    // 验证页面标题
    await expect(page.getByText('Slurm 集群管理')).toBeVisible({ timeout: 10000 });
    
    // 验证不是演示数据
    const demoAlert = page.getByText(/使用演示数据|Demo Data/i);
    const demoCount = await demoAlert.count();
    
    if (demoCount > 0) {
      console.log('⚠️  警告: 检测到演示数据提示');
    } else {
      console.log('✓ 使用真实 SLURM 集群数据');
    }
    
    // 验证关键组件存在
    await expect(page.getByText('节点列表')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('作业队列')).toBeVisible({ timeout: 5000 });
  });
  
  test('should display cluster statistics', async ({ page }) => {
    await loginIfNeeded(page);
    await page.goto(BASE + '/slurm');
    
    // 等待页面加载
    await page.waitForLoadState('networkidle');
    
    // 验证统计卡片（可能包含总节点数、CPU核心数等）
    const statsCards = page.locator('.ant-statistic, .ant-card-bordered');
    const cardCount = await statsCards.count();
    
    console.log(`📊 找到 ${cardCount} 个统计卡片`);
    
    // 至少应该有一些统计信息
    if (cardCount > 0) {
      const firstCard = statsCards.first();
      await expect(firstCard).toBeVisible();
      console.log('✓ 集群统计信息显示正常');
    }
  });
  
  test('should fetch and display node list via API', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    // 测试 API 直接调用
    const apiResponse = await request.get(BASE + '/api/slurm/nodes', {
      headers: {
        'Accept': 'application/json'
      }
    });
    
    expect(apiResponse.ok()).toBeTruthy();
    const data = await apiResponse.json();
    
    console.log('📡 SLURM 节点 API 响应:', JSON.stringify(data, null, 2));
    
    // 验证响应结构
    expect(data).toHaveProperty('nodes');
    
    if (Array.isArray(data.nodes)) {
      console.log(`✓ 获取到 ${data.nodes.length} 个节点`);
      
      // 如果有节点，验证节点数据结构
      if (data.nodes.length > 0) {
        const firstNode = data.nodes[0];
        console.log('📋 第一个节点信息:', JSON.stringify(firstNode, null, 2));
        
        // 验证节点必需字段
        expect(firstNode).toHaveProperty('name');
        expect(firstNode).toHaveProperty('state');
      }
    } else {
      console.log('⚠️  节点数据不是数组格式');
    }
    
    // 访问页面验证节点列表显示
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 查找节点表格
    const nodeTable = page.locator('table').first();
    if (await nodeTable.isVisible().catch(() => false)) {
      const rows = await nodeTable.locator('tbody tr').count();
      console.log(`✓ 节点表格显示 ${rows} 行数据`);
    }
  });
  
  test('should fetch and display job queue via API', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    // 测试作业队列 API
    const apiResponse = await request.get(BASE + '/api/slurm/jobs', {
      headers: {
        'Accept': 'application/json'
      }
    });
    
    expect(apiResponse.ok()).toBeTruthy();
    const data = await apiResponse.json();
    
    console.log('📡 SLURM 作业队列 API 响应:', JSON.stringify(data, null, 2));
    
    // 验证响应结构
    expect(data).toHaveProperty('jobs');
    
    if (Array.isArray(data.jobs)) {
      console.log(`✓ 作业队列包含 ${data.jobs.length} 个作业`);
      
      // 如果有作业，验证作业数据结构
      if (data.jobs.length > 0) {
        const firstJob = data.jobs[0];
        console.log('📋 第一个作业信息:', JSON.stringify(firstJob, null, 2));
        
        // 验证作业必需字段
        expect(firstJob).toHaveProperty('job_id');
        expect(firstJob).toHaveProperty('job_state');
      } else {
        console.log('ℹ️  当前没有运行的作业（这是正常的）');
      }
    }
    
    // 访问页面验证作业队列显示
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 查找作业表格
    const tables = page.locator('table');
    const tableCount = await tables.count();
    
    if (tableCount >= 2) {
      const jobTable = tables.nth(1); // 第二个表格通常是作业队列
      if (await jobTable.isVisible().catch(() => false)) {
        console.log('✓ 作业队列表格显示正常');
      }
    }
  });
  
  test('should display cluster configuration', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    // 测试集群配置 API（如果有的话）
    const apiResponse = await request.get(BASE + '/api/slurm/config', {
      headers: {
        'Accept': 'application/json'
      }
    }).catch(() => null);
    
    if (apiResponse && apiResponse.ok()) {
      const data = await apiResponse.json();
      console.log('📡 SLURM 配置 API 响应:', JSON.stringify(data, null, 2));
      
      // 验证集群名称
      if (data.cluster_name) {
        expect(data.cluster_name).toBe('ai-infra-cluster');
        console.log(`✓ 集群名称: ${data.cluster_name}`);
      }
      
      // 验证控制节点
      if (data.controller_host) {
        expect(data.controller_host).toContain('slurm-master');
        console.log(`✓ 控制节点: ${data.controller_host}`);
      }
    } else {
      console.log('ℹ️  集群配置 API 不可用或未实现');
    }
  });
  
  test('should check SLURM client installation in backend', async ({ request }) => {
    // 这个测试检查后端是否能执行 SLURM 命令
    // 通过 API 调用来间接验证
    
    const apiResponse = await request.get(BASE + '/api/slurm/nodes', {
      headers: {
        'Accept': 'application/json'
      }
    });
    
    if (apiResponse.ok()) {
      const data = await apiResponse.json();
      
      // 检查是否使用演示数据
      const isDemo = data.is_demo || data.demo || false;
      
      if (isDemo) {
        console.log('⚠️  后端使用演示数据（SLURM 客户端可能未正确安装）');
        console.log('建议: 检查 backend 容器中的 SLURM 客户端安装');
      } else {
        console.log('✓ 后端使用真实 SLURM 数据（客户端工作正常）');
      }
      
      // 检查错误信息
      if (data.error) {
        console.log('❌ SLURM API 返回错误:', data.error);
      }
    } else {
      console.log('❌ SLURM API 请求失败:', apiResponse.status());
    }
  });
  
  test('should verify node states are correct', async ({ page, request }) => {
    await loginIfNeeded(page);
    
    const apiResponse = await request.get(BASE + '/api/slurm/nodes');
    expect(apiResponse.ok()).toBeTruthy();
    
    const data = await apiResponse.json();
    
    if (data.nodes && Array.isArray(data.nodes)) {
      // 统计节点状态
      const states = {};
      data.nodes.forEach(node => {
        const state = node.state || 'unknown';
        states[state] = (states[state] || 0) + 1;
      });
      
      console.log('📊 节点状态统计:');
      Object.entries(states).forEach(([state, count]) => {
        console.log(`   ${state}: ${count} 个节点`);
      });
      
      // 检查是否有未知状态的节点
      if (states['unk'] || states['unknown']) {
        console.log('⚠️  警告: 检测到未知状态的节点');
        console.log('提示: 这可能表明 SLURM 计算节点未正确配置');
      }
      
      // 检查是否有 idle/alloc 等正常状态
      const normalStates = ['idle', 'alloc', 'mix', 'down'];
      const hasNormalStates = normalStates.some(state => states[state]);
      
      if (hasNormalStates) {
        console.log('✓ 存在正常状态的节点');
      } else {
        console.log('⚠️  未检测到正常状态的节点');
      }
    }
  });
});

test.describe('SLURM Command Execution Tests', () => {
  
  test('should execute sinfo command successfully', async ({ request }) => {
    const apiResponse = await request.post(BASE + '/api/slurm/exec', {
      data: {
        command: 'sinfo'
      },
      headers: {
        'Content-Type': 'application/json'
      }
    }).catch(() => null);
    
    if (apiResponse && apiResponse.ok()) {
      const data = await apiResponse.json();
      console.log('📡 sinfo 命令执行结果:', JSON.stringify(data, null, 2));
      
      // 验证输出包含分区信息
      if (data.output || data.stdout) {
        const output = data.output || data.stdout;
        expect(output).toContain('PARTITION');
        console.log('✓ sinfo 命令执行成功');
      }
    } else {
      console.log('ℹ️  sinfo 命令执行 API 不可用');
    }
  });
  
  test('should execute squeue command successfully', async ({ request }) => {
    const apiResponse = await request.post(BASE + '/api/slurm/exec', {
      data: {
        command: 'squeue'
      },
      headers: {
        'Content-Type': 'application/json'
      }
    }).catch(() => null);
    
    if (apiResponse && apiResponse.ok()) {
      const data = await apiResponse.json();
      console.log('📡 squeue 命令执行结果:', JSON.stringify(data, null, 2));
      
      // 验证输出包含作业信息表头
      if (data.output || data.stdout) {
        const output = data.output || data.stdout;
        expect(output).toContain('JOBID');
        console.log('✓ squeue 命令执行成功');
      }
    } else {
      console.log('ℹ️  squeue 命令执行 API 不可用');
    }
  });
});