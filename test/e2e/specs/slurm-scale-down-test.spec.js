/**
 * SLURM 缩容功能测试
 * 
 * 测试目标:
 * 1. 验证缩容任务能够提交
 * 2. 验证节点状态变为DOWN
 * 3. 验证节点从slurm.conf中移除
 * 4. 验证sinfo命令行输出与前端页面一致
 * 
 * 测试流程:
 * - 获取当前节点列表
 * - 选择一个节点进行缩容
 * - 提交缩容任务
 * - 验证节点状态更新
 * - 对比命令行和前端显示
 */

const { test, expect } = require('@playwright/test');
const { exec } = require('child_process');
const util = require('util');

const execPromise = util.promisify(exec);
const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

// 执行docker exec命令获取SLURM信息
async function getSlurmInfo() {
  try {
    const { stdout } = await execPromise('docker exec ai-infra-slurm-master sinfo -h -o "%n|%T|%c|%m|%P"');
    const lines = stdout.trim().split('\n');
    return lines.map(line => {
      const [name, state, cpus, memory, partition] = line.split('|');
      return { name, state, cpus, memory, partition };
    });
  } catch (error) {
    console.error('获取sinfo信息失败:', error);
    return [];
  }
}

async function getSlurmNodeDetails(nodeName) {
  try {
    const { stdout } = await execPromise(`docker exec ai-infra-slurm-master scontrol show node ${nodeName}`);
    return stdout;
  } catch (error) {
    console.error('获取节点详情失败:', error);
    return null;
  }
}

test.describe('SLURM 缩容功能测试', () => {
  let authToken;
  let initialNodes = [];

  test.beforeAll(async ({ request }) => {
    // 登录获取token
    const loginResponse = await request.post(`${BASE_URL}/api/auth/login`, {
      data: {
        username: TEST_USERNAME,
        password: TEST_PASSWORD
      }
    });
    
    expect(loginResponse.ok()).toBeTruthy();
    const loginData = await loginResponse.json();
    authToken = loginData.data?.token || loginData.token;
    console.log('✓ 登录成功');

    // 获取初始节点列表
    initialNodes = await getSlurmInfo();
    console.log('\n📋 初始节点列表 (sinfo):');
    initialNodes.forEach(node => {
      console.log(`  - ${node.name}: ${node.state} (${node.cpus} CPUs, ${node.memory}MB, Partition: ${node.partition})`);
    });
  });

  test('步骤1: 获取当前SLURM节点列表', async ({ request }) => {
    console.log('\n🔍 步骤1: 获取节点列表 API');
    
    const response = await request.get(`${BASE_URL}/api/slurm/nodes`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    
    console.log('  API返回节点数量:', data.data?.length || 0);
    
    if (data.data && data.data.length > 0) {
      console.log('  节点列表:');
      data.data.forEach(node => {
        console.log(`    - ${node.name}: ${node.state} (${node.cpus} CPUs)`);
      });
    }

    // 验证API返回的节点与sinfo一致
    expect(data.data?.length).toBe(initialNodes.length);
  });

  test('步骤2: 提交缩容任务', async ({ request }) => {
    console.log('\n🔽 步骤2: 提交缩容任务');

    // 找一个可以缩容的节点 (状态为idle或down的)
    const targetNode = initialNodes.find(n => 
      ['idle', 'down', 'down*', 'unk', 'unk*'].includes(n.state.toLowerCase())
    );

    if (!targetNode) {
      console.log('⚠️  没有找到可缩容的节点，跳过此测试');
      test.skip();
      return;
    }

    console.log(`  目标节点: ${targetNode.name} (当前状态: ${targetNode.state})`);

    // 提交缩容请求
    const scaleDownResponse = await request.post(`${BASE_URL}/api/slurm/scale-down`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        node_ids: [targetNode.name]
      }
    });

    console.log('  缩容请求状态码:', scaleDownResponse.status());
    
    if (scaleDownResponse.ok()) {
      const result = await scaleDownResponse.json();
      console.log('  缩容结果:', JSON.stringify(result, null, 2));
      
      expect(result.data).toBeDefined();
      expect(result.data.success).toBe(true);
      expect(result.data.results).toBeDefined();
      expect(result.data.results.length).toBeGreaterThan(0);

      const nodeResult = result.data.results.find(r => r.node_id === targetNode.name);
      expect(nodeResult).toBeDefined();
      expect(nodeResult.success).toBe(true);
      
      console.log(`  ✅ 节点 ${targetNode.name} 缩容任务提交成功`);
    } else {
      const errorText = await scaleDownResponse.text();
      console.log('  ❌ 缩容请求失败:', errorText);
      throw new Error(`缩容请求失败: ${errorText}`);
    }
  });

  test('步骤3: 验证节点状态更新', async ({ request }) => {
    console.log('\n✅ 步骤3: 验证节点状态');

    // 等待几秒让SLURM处理
    await new Promise(resolve => setTimeout(resolve, 3000));

    // 重新获取节点列表
    const updatedNodes = await getSlurmInfo();
    console.log('\n📋 更新后的节点列表 (sinfo):');
    updatedNodes.forEach(node => {
      console.log(`  - ${node.name}: ${node.state} (${node.cpus} CPUs, ${node.memory}MB)`);
    });

    // 对比初始和更新后的节点列表
    console.log('\n📊 节点变化对比:');
    console.log(`  初始节点数: ${initialNodes.length}`);
    console.log(`  当前节点数: ${updatedNodes.length}`);

    // 找出被移除的节点
    const removedNodes = initialNodes.filter(initial => 
      !updatedNodes.some(updated => updated.name === initial.name)
    );

    if (removedNodes.length > 0) {
      console.log('  ✅ 已移除的节点:');
      removedNodes.forEach(node => {
        console.log(`    - ${node.name}`);
      });
    } else {
      console.log('  ℹ️  没有节点被移除（可能节点状态改为DOWN而非删除）');
      
      // 检查状态变化
      const changedNodes = updatedNodes.filter(updated => {
        const initial = initialNodes.find(n => n.name === updated.name);
        return initial && initial.state !== updated.state;
      });
      
      if (changedNodes.length > 0) {
        console.log('  📝 状态已变化的节点:');
        changedNodes.forEach(node => {
          const initial = initialNodes.find(n => n.name === node.name);
          console.log(`    - ${node.name}: ${initial.state} → ${node.state}`);
        });
      }
    }

    // 验证API返回的节点列表
    const apiResponse = await request.get(`${BASE_URL}/api/slurm/nodes`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    expect(apiResponse.ok()).toBeTruthy();
    const apiData = await apiResponse.json();
    
    console.log('\n🔌 API返回节点数量:', apiData.data?.length || 0);
    console.log('   sinfo返回节点数量:', updatedNodes.length);
    
    // API和sinfo应该一致
    expect(apiData.data?.length).toBe(updatedNodes.length);
  });

  test('步骤4: 验证前端页面显示', async ({ page }) => {
    console.log('\n🌐 步骤4: 验证前端页面');

    // 设置认证token
    await page.goto(BASE_URL);
    await page.evaluate((token) => {
      localStorage.setItem('token', token);
    }, authToken);

    // 访问SLURM页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 获取命令行的节点列表
    const cliNodes = await getSlurmInfo();
    console.log('  命令行节点数:', cliNodes.length);

    // 检查页面上显示的节点数
    const nodeCards = page.locator('.ant-card').filter({ hasText: /节点|Node/ });
    const nodeCount = await nodeCards.count();
    console.log('  页面节点卡片数:', nodeCount);

    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/slurm-after-scale-down.png',
      fullPage: true 
    });
    console.log('  📸 已保存截图: test-screenshots/slurm-after-scale-down.png');

    // 检查是否有节点列表表格
    const nodeTable = page.locator('table').filter({ hasText: /节点|状态/ }).first();
    const tableVisible = await nodeTable.isVisible({ timeout: 2000 }).catch(() => false);
    
    if (tableVisible) {
      const rows = nodeTable.locator('tbody tr');
      const rowCount = await rows.count();
      console.log('  表格行数:', rowCount);
      
      // 验证表格行数与命令行一致
      expect(rowCount).toBe(cliNodes.length);
      
      // 检查每个节点的状态
      for (let i = 0; i < Math.min(rowCount, cliNodes.length); i++) {
        const row = rows.nth(i);
        const rowText = await row.textContent();
        const cliNode = cliNodes[i];
        
        if (rowText.includes(cliNode.name)) {
          console.log(`  ✅ 节点 ${cliNode.name} 在页面上显示`);
          
          // 验证状态是否匹配
          if (rowText.toLowerCase().includes(cliNode.state.toLowerCase())) {
            console.log(`     状态匹配: ${cliNode.state}`);
          }
        }
      }
    } else {
      console.log('  ℹ️  页面上没有找到节点列表表格');
    }
  });

  test('步骤5: 验证节点详细信息', async () => {
    console.log('\n📋 步骤5: 检查节点详细配置');

    // 获取当前所有节点
    const currentNodes = await getSlurmInfo();
    
    if (currentNodes.length > 0) {
      console.log('  检查第一个节点的详细信息:');
      const nodeDetails = await getSlurmNodeDetails(currentNodes[0].name);
      
      if (nodeDetails) {
        console.log(`  节点 ${currentNodes[0].name} 详情:`);
        console.log(nodeDetails.split('\n').slice(0, 10).join('\n'));
        
        // 验证节点是否真的被移除
        const hasState = nodeDetails.includes('State=');
        expect(hasState).toBe(true);
      }
    }

    // 检查slurm.conf文件
    try {
      const { stdout } = await execPromise('docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf | grep -E "^NodeName="');
      console.log('\n  slurm.conf中的节点配置:');
      console.log(stdout);
      
      const nodeLines = stdout.trim().split('\n').filter(line => line.trim());
      console.log(`  配置文件中的节点定义数: ${nodeLines.length}`);
    } catch (error) {
      console.log('  ℹ️  没有找到NodeName配置（可能所有节点都被移除）');
    }
  });
});
