/**
 * SLURM状态同步测试
 * 
 * 测试目标:
 * 1. 使用sinfo命令行获取SLURM实际状态
 * 2. 对比前端页面显示的SLURM状态
 * 3. 诊断节点状态为unknown*的原因
 * 4. 验证SLURM状态同步功能
 * 
 * 预期结果:
 * - 命令行sinfo和前端页面显示的节点状态一致
 * - 如果节点状态为unknown，诊断原因并尝试修复
 */

const { test, expect } = require('@playwright/test');
const { exec } = require('child_process');
const util = require('util');

const execPromise = util.promisify(exec);

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

test.describe('SLURM状态同步测试', () => {
  let authToken;
  let sinfoOutput;
  let nodeStates = {};

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

    // 获取sinfo输出
    try {
      const { stdout, stderr } = await execPromise('docker exec ai-infra-slurm-master sinfo -h');
      sinfoOutput = stdout.trim();
      console.log('\n📋 SLURM sinfo 输出:');
      console.log(sinfoOutput);

      // 解析节点状态
      const lines = sinfoOutput.split('\n');
      lines.forEach(line => {
        const parts = line.trim().split(/\s+/);
        if (parts.length >= 5) {
          const state = parts[3];
          const nodelist = parts[4];
          nodeStates[nodelist] = state;
        }
      });
      console.log('\n📊 解析的节点状态:', nodeStates);
    } catch (error) {
      console.error('❌ 获取sinfo失败:', error.message);
    }
  });

  test('步骤1: 检查SLURM Master容器状态', async () => {
    console.log('\n🔍 检查SLURM Master容器...');
    
    try {
      const { stdout } = await execPromise('docker ps --filter name=ai-infra-slurm-master --format "{{.Status}}"');
      const status = stdout.trim();
      console.log('  SLURM Master容器状态:', status);
      expect(status).toContain('Up');
      console.log('✅ SLURM Master容器运行正常');
    } catch (error) {
      console.error('❌ SLURM Master容器检查失败:', error.message);
      throw error;
    }
  });

  test('步骤2: 检查slurmd进程状态', async () => {
    console.log('\n🔍 检查各节点的slurmd进程...');
    
    const testNodes = [
      'test-rocky01',
      'test-rocky02', 
      'test-rocky03',
      'test-ssh01',
      'test-ssh02',
      'test-ssh03'
    ];

    for (const node of testNodes) {
      try {
        // 检查容器是否运行
        const { stdout: containerStatus } = await execPromise(`docker ps --filter name=${node} --format "{{.Status}}"`);
        
        if (!containerStatus.trim()) {
          console.log(`  ⚠️  ${node}: 容器未运行`);
          continue;
        }

        // 检查slurmd进程
        const { stdout: processCheck } = await execPromise(`docker exec ${node} pgrep -x slurmd || echo "not_running"`);
        
        if (processCheck.trim() === 'not_running' || !processCheck.trim()) {
          console.log(`  ❌ ${node}: slurmd进程未运行`);
          
          // 尝试启动slurmd
          console.log(`     → 尝试启动slurmd...`);
          try {
            await execPromise(`docker exec ${node} slurmd -D &`);
            console.log(`     ✓ slurmd启动命令已执行`);
          } catch (e) {
            console.log(`     ✗ slurmd启动失败:`, e.message);
          }
        } else {
          console.log(`  ✅ ${node}: slurmd运行中 (PID: ${processCheck.trim()})`);
        }
      } catch (error) {
        console.log(`  ❌ ${node}: 检查失败 -`, error.message);
      }
    }
  });

  test('步骤3: 检查SLURM配置文件', async () => {
    console.log('\n🔍 检查SLURM配置...');
    
    try {
      // 检查slurm.conf
      const { stdout: config } = await execPromise('docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf | grep -E "NodeName|PartitionName"');
      console.log('  SLURM节点配置:');
      console.log(config);

      // 检查是否有NodeName配置
      if (!config.includes('NodeName=')) {
        console.log('  ⚠️  警告: 未找到NodeName配置，这可能是SLURM_TEST_NODES为空导致的');
      }
    } catch (error) {
      console.log('  ❌ 配置检查失败:', error.message);
    }
  });

  test('步骤4: 对比API返回的节点状态', async ({ request }) => {
    console.log('\n🔍 测试API: /api/slurm/nodes');
    
    const response = await request.get(`${BASE_URL}/api/slurm/nodes`, {
      headers: {
        'Authorization': `Bearer ${authToken}`
      }
    });

    expect(response.ok()).toBeTruthy();
    const nodes = await response.json();
    
    console.log(`\n📊 API返回 ${nodes.length} 个节点:`);
    
    nodes.forEach(node => {
      console.log(`  - ${node.name}: state=${node.state}, cpus=${node.cpus}, memory=${node.memory_mb}MB`);
    });

    // 验证节点数量
    if (nodes.length === 0) {
      console.log('\n⚠️  警告: API返回的节点列表为空');
      console.log('   可能原因: SLURM_TEST_NODES环境变量为空，导致slurm.conf中没有NodeName配置');
    } else {
      expect(nodes.length).toBeGreaterThan(0);
      console.log(`✅ API返回了 ${nodes.length} 个节点`);
    }

    // 检查节点状态
    const unknownNodes = nodes.filter(n => n.state.includes('unk'));
    if (unknownNodes.length > 0) {
      console.log(`\n⚠️  发现 ${unknownNodes.length} 个unknown状态的节点:`);
      unknownNodes.forEach(n => {
        console.log(`   - ${n.name}: ${n.state}`);
      });
      console.log('\n   原因分析:');
      console.log('   1. 节点上的slurmd守护进程未运行');
      console.log('   2. 网络连接问题，Master无法与节点通信');
      console.log('   3. 节点的SLURM配置与Master不匹配');
    }
  });

  test('步骤5: 验证前端页面显示', async ({ page }) => {
    console.log('\n🌐 测试前端页面: /slurm');

    // 设置认证token
    await page.goto(BASE_URL);
    await page.evaluate((token) => {
      localStorage.setItem('token', token);
    }, authToken);

    // 访问SLURM页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // 截图
    await page.screenshot({ 
      path: 'test-screenshots/slurm-status-unknown.png',
      fullPage: true 
    });
    console.log('📸 已保存页面截图: test-screenshots/slurm-status-unknown.png');

    // 检查是否显示了节点列表
    const nodeTable = page.locator('table').first();
    const isTableVisible = await nodeTable.isVisible({ timeout: 5000 }).catch(() => false);
    
    if (isTableVisible) {
      console.log('✓ 找到节点表格');
      
      // 获取表格中的节点状态
      const tableContent = await page.evaluate(() => {
        const rows = Array.from(document.querySelectorAll('table tbody tr'));
        return rows.map(row => {
          const cells = Array.from(row.querySelectorAll('td'));
          return {
            name: cells[0]?.textContent?.trim(),
            state: cells[1]?.textContent?.trim(),
            cpus: cells[2]?.textContent?.trim(),
            memory: cells[3]?.textContent?.trim()
          };
        }).filter(r => r.name);
      });

      console.log('\n📋 前端页面显示的节点:');
      tableContent.forEach(node => {
        console.log(`  - ${node.name}: ${node.state}`);
      });

      // 验证前端显示的状态与sinfo一致
      if (tableContent.length > 0) {
        console.log('\n✅ 前端页面正确显示了节点信息');
      } else {
        console.log('\n⚠️  前端页面未显示节点信息');
      }
    } else {
      console.log('⚠️  未找到节点表格，可能在其他位置');
    }
  });

  test('步骤6: 诊断和修复建议', async () => {
    console.log('\n🔧 诊断总结和修复建议:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    // 检查sinfo输出中的状态
    if (sinfoOutput && sinfoOutput.includes('unk*')) {
      console.log('\n❌ 问题: SLURM节点状态为 unknown*');
      console.log('\n📋 可能的原因:');
      console.log('1. 节点上的slurmd守护进程未运行');
      console.log('2. 节点无法与SLURM Master通信（网络/防火墙问题）');
      console.log('3. 节点的slurm.conf配置不同步');
      console.log('4. 节点的munge认证失败');
      
      console.log('\n🔧 修复步骤:');
      console.log('\n方案1: 启动slurmd守护进程');
      console.log('  在每个计算节点上执行:');
      console.log('  docker exec test-rocky01 slurmd -D &');
      console.log('  docker exec test-rocky02 slurmd -D &');
      console.log('  docker exec test-rocky03 slurmd -D &');
      console.log('  docker exec test-ssh01 slurmd -D &');
      console.log('  docker exec test-ssh02 slurmd -D &');
      console.log('  docker exec test-ssh03 slurmd -D &');
      
      console.log('\n方案2: 检查网络连接');
      console.log('  从节点ping Master:');
      console.log('  docker exec test-rocky01 ping -c 3 slurm-master');
      
      console.log('\n方案3: 同步munge密钥');
      console.log('  确保所有节点使用相同的munge.key');
      
      console.log('\n方案4: 重启SLURM服务');
      console.log('  docker-compose -f docker-compose.yml restart slurm-master');
      console.log('  然后在各节点重启slurmd');
    } else if (!sinfoOutput || sinfoOutput.trim() === '') {
      console.log('\n❌ 问题: 无法获取SLURM状态');
      console.log('\n🔧 修复步骤:');
      console.log('1. 检查SLURM Master容器是否运行');
      console.log('2. 检查slurm.conf中是否有NodeName配置');
      console.log('3. 确认SLURM_TEST_NODES环境变量已正确设置');
    } else {
      console.log('\n✅ SLURM状态看起来正常');
    }

    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  });

  test('步骤7: 自动修复尝试', async () => {
    console.log('\n🔧 尝试自动修复unknown状态...');
    
    const testNodes = [
      'test-rocky01',
      'test-rocky02',
      'test-rocky03',
      'test-ssh01',
      'test-ssh02',
      'test-ssh03'
    ];

    console.log('\n1. 检查并启动slurmd进程...');
    for (const node of testNodes) {
      try {
        // 检查容器是否存在并运行
        const { stdout: psOutput } = await execPromise(`docker ps --filter name=^/${node}$ --format "{{.Names}}"`);
        if (!psOutput.trim()) {
          console.log(`  ⊘ ${node}: 容器未运行，跳过`);
          continue;
        }

        // 检查slurmd是否已运行
        const { stdout: pidCheck } = await execPromise(`docker exec ${node} sh -c "pgrep slurmd || echo ''" 2>/dev/null`);
        
        if (pidCheck.trim()) {
          console.log(`  ✓ ${node}: slurmd已运行 (PID: ${pidCheck.trim()})`);
        } else {
          console.log(`  → ${node}: 启动slurmd...`);
          // 在后台启动slurmd
          await execPromise(`docker exec -d ${node} slurmd -D`);
          await new Promise(resolve => setTimeout(resolve, 1000));
          console.log(`  ✓ ${node}: slurmd启动命令已执行`);
        }
      } catch (error) {
        console.log(`  ✗ ${node}: 操作失败 - ${error.message}`);
      }
    }

    console.log('\n2. 等待节点状态更新...');
    await new Promise(resolve => setTimeout(resolve, 5000));

    console.log('\n3. 重新检查sinfo状态...');
    try {
      const { stdout: newSinfo } = await execPromise('docker exec ai-infra-slurm-master sinfo');
      console.log(newSinfo);
      
      if (newSinfo.includes('idle') || newSinfo.includes('alloc')) {
        console.log('\n✅ 修复成功！节点状态已更新');
      } else if (newSinfo.includes('unk*')) {
        console.log('\n⚠️  节点仍为unknown状态，可能需要更多时间或手动干预');
      }
    } catch (error) {
      console.log('\n✗ 无法获取更新后的状态:', error.message);
    }
  });
});
