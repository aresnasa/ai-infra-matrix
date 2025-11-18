// @ts-nocheck
/* eslint-disable */
/**
 * SLURM Node Down Status Diagnosis Test
 * 
 * 目的: 诊断并修复 SLURM 节点 down* 状态问题
 * 
 * 测试内容:
 * 1. 验证当前节点状态
 * 2. 检查节点配置
 * 3. 诊断 down* 原因
 * 4. 验证 SLURM REST API 部署
 */

const { test, expect } = require('@playwright/test');

const BASE = process.env.BASE_URL || 'http://192.168.0.200:8080';

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

test.describe('SLURM Node Down Status Diagnosis', () => {
  
  test('should display current node status and identify down nodes', async ({ page }) => {
    await loginIfNeeded(page);
    
    // 导航到 SLURM 管理页面
    await page.goto(BASE + '/slurm');
    await page.waitForLoadState('networkidle');
    
    // 截图当前状态
    await page.screenshot({ 
      path: 'test-screenshots/slurm-nodes-down-status.png',
      fullPage: true 
    });
    
    console.log('📸 已保存 SLURM 页面截图');
    
    // 检查节点表格
    const nodeTable = page.locator('table').first();
    if (await nodeTable.isVisible()) {
      // 获取所有行
      const rows = nodeTable.locator('tbody tr');
      const rowCount = await rows.count();
      
      console.log(`\n📊 节点列表 (共 ${rowCount} 个节点):`);
      console.log('─'.repeat(80));
      
      // 遍历每一行
      for (let i = 0; i < rowCount; i++) {
        const row = rows.nth(i);
        const cells = row.locator('td');
        const cellCount = await cells.count();
        
        if (cellCount >= 3) {
          const nodeName = await cells.nth(0).textContent();
          const partition = await cells.nth(1).textContent();
          const state = await cells.nth(2).textContent();
          const cpus = cellCount > 3 ? await cells.nth(3).textContent() : 'N/A';
          const memory = cellCount > 4 ? await cells.nth(4).textContent() : 'N/A';
          
          const stateIcon = state.includes('down') ? '❌' : 
                          state.includes('idle') ? '✅' : 
                          state.includes('alloc') ? '🟢' : '⚠️';
          
          console.log(`${stateIcon} ${nodeName.trim()}\t${partition.trim()}\t${state.trim()}\t${cpus.trim()}\t${memory.trim()}`);
        }
      }
      console.log('─'.repeat(80));
      
      // 检查是否有 down 状态的节点
      const pageText = await page.textContent('body');
      if (pageText.includes('down')) {
        console.log('\n⚠️  检测到 down* 状态的节点');
        console.log('可能原因:');
        console.log('  1. 计算节点未安装 slurmd');
        console.log('  2. slurmd 服务未启动');
        console.log('  3. 节点网络不通');
        console.log('  4. munge 认证失败');
      }
    }
  });
  
  test('should check node details via API', async ({ request }) => {
    // 获取登录 token
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    expect(loginResponse.ok()).toBeTruthy();
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    // 获取节点列表
    const nodesResponse = await request.get(BASE + '/api/slurm/nodes', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    expect(nodesResponse.ok()).toBeTruthy();
    const nodesData = await nodesResponse.json();
    
    console.log('\n📡 API 返回的节点数据:');
    console.log(JSON.stringify(nodesData, null, 2));
    
    if (nodesData.data && Array.isArray(nodesData.data)) {
      const nodes = nodesData.data;
      
      console.log(`\n总节点数: ${nodes.length}`);
      console.log(`Demo 模式: ${nodesData.demo ? '是' : '否'}`);
      
      // 统计状态
      const downNodes = nodes.filter(n => n.state && n.state.includes('down'));
      const idleNodes = nodes.filter(n => n.state && n.state.includes('idle'));
      const allocNodes = nodes.filter(n => n.state && n.state.includes('alloc'));
      
      console.log(`\n状态统计:`);
      console.log(`  ❌ Down: ${downNodes.length}`);
      console.log(`  ✅ Idle: ${idleNodes.length}`);
      console.log(`  🟢 Alloc: ${allocNodes.length}`);
      
      if (downNodes.length > 0) {
        console.log(`\n❌ Down 节点详情:`);
        downNodes.forEach(node => {
          console.log(`  - ${node.name}: ${node.state}`);
        });
      }
    }
  });
  
  test('should check SLURM master sinfo output', async ({ request }) => {
    const loginResponse = await request.post(BASE + '/api/auth/login', {
      data: {
        username: 'admin',
        password: 'admin123'
      }
    });
    
    const loginData = await loginResponse.json();
    const token = loginData.token;
    
    // 尝试调用 SLURM 命令执行 API（如果存在）
    const execResponse = await request.post(BASE + '/api/slurm/exec', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: {
        command: 'sinfo'
      }
    }).catch(() => null);
    
    if (execResponse && execResponse.ok()) {
      const data = await execResponse.json();
      console.log('\n📡 sinfo 命令输出:');
      console.log(data.output || data.stdout || JSON.stringify(data));
    } else {
      console.log('\nℹ️  SLURM exec API 尚未实现，需要添加此功能');
    }
  });
  
  test('should verify expected nodes are registered', async ({ request }) => {
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
    
    const nodesData = await nodesResponse.json();
    
    if (nodesData.data && Array.isArray(nodesData.data)) {
      const expectedNodes = ['test-ssh01', 'test-ssh02', 'test-ssh03'];
      const actualNodes = nodesData.data.map(n => n.name);
      
      console.log('\n✓ 预期节点:', expectedNodes);
      console.log('✓ 实际节点:', actualNodes);
      
      expectedNodes.forEach(nodeName => {
        const found = actualNodes.includes(nodeName);
        if (found) {
          console.log(`  ✅ ${nodeName} - 已注册`);
        } else {
          console.log(`  ❌ ${nodeName} - 未找到`);
        }
      });
      
      // 验证所有预期节点都存在
      expectedNodes.forEach(nodeName => {
        expect(actualNodes).toContain(nodeName);
      });
    }
  });
});

test.describe('SLURM REST API Tests', () => {
  
  test('should check if SLURM REST API is available', async ({ request }) => {
    // 尝试访问 SLURM REST API (通常在端口 6820)
    const restApiUrl = 'http://192.168.0.200:6820/slurm/v0.0.40/diag';
    
    const response = await request.get(restApiUrl, {
      failOnStatusCode: false
    }).catch(() => null);
    
    if (response && response.ok()) {
      const data = await response.json();
      console.log('\n✅ SLURM REST API 可用');
      console.log('诊断信息:', JSON.stringify(data, null, 2));
    } else {
      console.log('\n⚠️  SLURM REST API 不可用');
      console.log('需要部署 slurmrestd 服务');
      console.log('建议步骤:');
      console.log('  1. 在 SLURM master 容器中安装 slurmrestd');
      console.log('  2. 配置 slurmrestd 监听端口 6820');
      console.log('  3. 暴露端口并测试连接');
    }
  });
  
  test('should test SLURM REST API nodes endpoint', async ({ request }) => {
    const restApiUrl = 'http://192.168.0.200:6820/slurm/v0.0.40/nodes';
    
    const response = await request.get(restApiUrl, {
      failOnStatusCode: false
    }).catch(() => null);
    
    if (response && response.ok()) {
      const data = await response.json();
      console.log('\n✅ SLURM REST API /nodes 端点可用');
      console.log('节点数据:', JSON.stringify(data, null, 2));
    } else {
      console.log('\n⚠️  SLURM REST API /nodes 端点不可用');
    }
  });
});
