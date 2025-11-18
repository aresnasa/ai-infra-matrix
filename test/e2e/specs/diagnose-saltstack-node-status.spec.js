const { test, expect } = require('@playwright/test');

/**
 * 诊断 SLURM 节点 SaltStack 状态显示问题
 * 
 * 目标：找出为什么节点显示"未配置"而不是真实的 minion 状态
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.91:8080';
const TEST_USERNAME = process.env.TEST_USERNAME || 'admin';
const TEST_PASSWORD = process.env.TEST_PASSWORD || 'admin123';

test.describe('诊断 SaltStack 节点状态', () => {
  let token = null;

  test.beforeAll(async ({ request }) => {
    // 获取认证 token
    const loginResponse = await request.post(`${BASE_URL}/api/auth/login`, {
      data: { username: TEST_USERNAME, password: TEST_PASSWORD }
    });
    
    if (loginResponse.ok()) {
      const loginData = await loginResponse.json();
      token = loginData.data?.token || loginData.token;
      console.log('✅ 登录成功');
    }
  });

  test('📊 完整诊断流程', async ({ request, page }) => {
    console.log('\n' + '='.repeat(70));
    console.log('  SaltStack 节点状态诊断');
    console.log('='.repeat(70));

    const headers = {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    };

    // ==================== 第1步: 检查 SaltStack 集成状态 ====================
    console.log('\n【第1步】检查 SaltStack 集成状态');
    console.log('-'.repeat(70));
    
    const saltIntegration = await request.get(`${BASE_URL}/api/slurm/saltstack/integration`, { headers });
    
    if (!saltIntegration.ok()) {
      console.log('❌ SaltStack 集成 API 失败');
      console.log('  状态码:', saltIntegration.status());
      return;
    }

    const saltData = await saltIntegration.json();
    const integration = saltData.data || saltData;
    
    console.log('✅ SaltStack 集成状态:');
    console.log('  Master 状态:', integration.master_status);
    console.log('  API 状态:', integration.api_status);
    console.log('  总 Minions:', integration.minions?.total || 0);
    console.log('  在线 Minions:', integration.minions?.online || 0);
    
    const minionIds = integration.minion_list?.map(m => m.id) || [];
    console.log('\n  Minion IDs:');
    minionIds.forEach(id => console.log(`    - ${id}`));

    // ==================== 第2步: 检查节点列表 API ====================
    console.log('\n【第2步】检查节点列表 API');
    console.log('-'.repeat(70));
    
    const nodesResponse = await request.get(`${BASE_URL}/api/slurm/nodes`, { headers });
    
    if (!nodesResponse.ok()) {
      console.log('❌ 节点列表 API 失败');
      console.log('  状态码:', nodesResponse.status());
      return;
    }

    const nodesData = await nodesResponse.json();
    const nodes = nodesData.data || [];
    
    console.log(`✅ 获取到 ${nodes.length} 个节点`);
    
    // ==================== 第3步: 分析节点状态 ====================
    console.log('\n【第3步】分析节点 SaltStack 状态');
    console.log('-'.repeat(70));
    
    const statusStats = {};
    const problemNodes = [];
    
    nodes.forEach(node => {
      const status = node.salt_status || 'undefined';
      statusStats[status] = (statusStats[status] || 0) + 1;
      
      if (status === 'unknown' || !node.salt_status) {
        problemNodes.push({
          name: node.name,
          salt_status: node.salt_status,
          salt_minion_id: node.salt_minion_id,
          salt_enabled: node.salt_enabled,
          salt_status_error: node.salt_status_error
        });
      }
    });

    console.log('📊 状态统计:');
    Object.entries(statusStats).forEach(([status, count]) => {
      const icon = status === 'accepted' ? '✅' : 
                   status === 'unknown' ? '⚠️' : 
                   status === 'undefined' ? '❌' : '📍';
      console.log(`  ${icon} ${status}: ${count} 个节点`);
    });

    // ==================== 第4步: 详细检查问题节点 ====================
    if (problemNodes.length > 0) {
      console.log('\n【第4步】问题节点详情');
      console.log('-'.repeat(70));
      console.log(`⚠️  发现 ${problemNodes.length} 个状态异常的节点:\n`);
      
      problemNodes.slice(0, 5).forEach((node, idx) => {
        console.log(`${idx + 1}. 节点: ${node.name}`);
        console.log(`   salt_status: ${node.salt_status || 'undefined'}`);
        console.log(`   salt_minion_id: ${node.salt_minion_id || 'undefined'}`);
        console.log(`   salt_enabled: ${node.salt_enabled}`);
        console.log(`   salt_status_error: ${node.salt_status_error || 'none'}`);
        
        // 检查是否匹配到 minion
        const matchedMinion = minionIds.find(id => 
          id === node.name || 
          id === node.salt_minion_id ||
          id.includes(node.name) ||
          node.name.includes(id)
        );
        
        if (matchedMinion) {
          console.log(`   ✅ 可以匹配到 minion: ${matchedMinion}`);
        } else {
          console.log(`   ❌ 无法匹配到任何 minion`);
          console.log(`   可用的 minions: ${minionIds.slice(0, 3).join(', ')}...`);
        }
        console.log('');
      });
    }

    // ==================== 第5步: 检查匹配逻辑 ====================
    console.log('\n【第5步】检查名称匹配逻辑');
    console.log('-'.repeat(70));
    
    console.log('节点名称 vs Minion ID 匹配分析:\n');
    
    nodes.slice(0, 5).forEach(node => {
      console.log(`节点: ${node.name}`);
      
      // 精确匹配
      if (minionIds.includes(node.name)) {
        console.log(`  ✅ 精确匹配: ${node.name}`);
      } else {
        // 短名称匹配
        const shortName = node.name.split('.')[0];
        if (minionIds.includes(shortName)) {
          console.log(`  ✅ 短名称匹配: ${shortName}`);
        } else {
          // 模糊匹配
          const fuzzyMatch = minionIds.find(id => 
            id.includes(node.name) || node.name.includes(id)
          );
          if (fuzzyMatch) {
            console.log(`  ⚠️  模糊匹配: ${fuzzyMatch}`);
          } else {
            console.log(`  ❌ 无匹配`);
            console.log(`     可能的 minions: ${minionIds.slice(0, 3).join(', ')}`);
          }
        }
      }
      console.log('');
    });

    // ==================== 第6步: 前端页面验证 ====================
    console.log('\n【第6步】前端页面显示验证');
    console.log('-'.repeat(70));
    
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    const usernameInput = page.locator('input[name="username"]').first();
    if (await usernameInput.isVisible({ timeout: 2000 })) {
      await usernameInput.fill(TEST_USERNAME);
      await page.locator('input[name="password"]').first().fill(TEST_PASSWORD);
      await page.locator('button[type="submit"]').first().click();
      await page.waitForURL(/\/(dashboard|slurm|home|projects)/i, { timeout: 10000 });
    }

    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 点击节点管理标签
    const nodesTab = page.locator('.ant-tabs-tab').filter({ hasText: /节点管理/ });
    if (await nodesTab.count() > 0) {
      await nodesTab.click();
      await page.waitForTimeout(1000);
    }

    // 检查表格中的状态显示
    const table = page.locator('.ant-table').first();
    const rows = table.locator('tbody tr');
    const rowCount = await rows.count();
    
    console.log(`\n前端显示 ${rowCount} 个节点:`);
    
    for (let i = 0; i < Math.min(rowCount, 5); i++) {
      const row = rows.nth(i);
      const cells = row.locator('td');
      
      if (await cells.count() > 0) {
        const nodeName = await cells.nth(0).innerText();
        
        // 查找 SaltStack 状态列
        let saltStatus = 'N/A';
        for (let j = 0; j < await cells.count(); j++) {
          const cellText = await cells.nth(j).innerText();
          if (cellText.includes('未配置') || cellText.includes('已连接') || 
              cellText.includes('待接受') || cellText.includes('已拒绝') ||
              cellText.includes('API 错误')) {
            saltStatus = cellText.trim();
            break;
          }
        }
        
        const icon = saltStatus.includes('已连接') ? '✅' : 
                     saltStatus.includes('未配置') ? '⚠️' : '❌';
        console.log(`  ${icon} ${nodeName}: ${saltStatus}`);
      }
    }

    await page.screenshot({ 
      path: 'test-screenshots/saltstack-node-status-diagnosis.png',
      fullPage: true 
    });
    console.log('\n📸 截图已保存: saltstack-node-status-diagnosis.png');

    // ==================== 诊断总结 ====================
    console.log('\n' + '='.repeat(70));
    console.log('  诊断总结');
    console.log('='.repeat(70));
    
    if (problemNodes.length === 0) {
      console.log('\n✅ 所有节点状态正常');
    } else {
      console.log('\n⚠️  发现问题:');
      console.log(`  - ${problemNodes.length} 个节点状态为 unknown 或 undefined`);
      
      if (integration.master_status !== 'running') {
        console.log('  - SaltStack Master 未运行');
      }
      
      const hasErrorNodes = problemNodes.some(n => n.salt_status_error);
      if (hasErrorNodes) {
        console.log('  - 部分节点有错误信息');
      }
      
      console.log('\n🔧 可能的原因:');
      console.log('  1. 节点名称与 Minion ID 不匹配');
      console.log('  2. Minion 未接受密钥');
      console.log('  3. 后端 enrichNodesWithSaltStackStatus 逻辑有问题');
      console.log('  4. SaltStack API 返回数据格式问题');
      
      console.log('\n💡 修复建议:');
      console.log('  1. 检查后端日志: docker logs backend | grep -i salt');
      console.log('  2. 验证 Minion 密钥: docker exec saltstack salt-key -L');
      console.log('  3. 测试连接: docker exec saltstack salt test-rocky01 test.ping');
      console.log('  4. 检查后端代码中的名称匹配逻辑');
    }
    
    console.log('\n' + '='.repeat(70));
  });
});
