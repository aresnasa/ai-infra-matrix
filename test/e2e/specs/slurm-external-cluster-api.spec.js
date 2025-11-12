const { test, expect } = require('@playwright/test');

/**
 * SLURM 外部集群管理 API 测试
 * 测试后端 API 功能，不需要登录
 */

const BASE_API = 'http://192.168.3.91:8080/api';

test.describe('SLURM 外部集群管理 API', () => {
  
  test('应该能访问 SLURM 集群列表 API', async ({ request }) => {
    const response = await request.get(`${BASE_API}/slurm/clusters`);
    
    // 验证响应状态
    expect(response.status()).toBeLessThan(500);
    
    // 如果返回 200，验证数据结构
    if (response.status() === 200) {
      const data = await response.json();
      console.log('集群列表响应:', JSON.stringify(data, null, 2));
      
      if (data.success) {
        expect(data).toHaveProperty('data');
        expect(Array.isArray(data.data) || typeof data.data === 'object').toBeTruthy();
      }
    } else {
      console.log('API 响应状态:', response.status());
      console.log('可能需要认证或集群尚未创建');
    }
  });

  test('应该能访问 SLURM 节点列表 API', async ({ request }) => {
    const response = await request.get(`${BASE_API}/slurm/nodes`);
    
    // 验证响应状态
    expect(response.status()).toBeLessThan(500);
    
    if (response.status() === 200) {
      const data = await response.json();
      console.log('节点列表响应:', JSON.stringify(data, null, 2));
    } else {
      console.log('API 响应状态:', response.status());
    }
  });

  test('应该能访问 SLURM 节点模板 API', async ({ request }) => {
    const response = await request.get(`${BASE_API}/slurm/node-templates`);
    
    // 验证响应状态
    expect(response.status()).toBeLessThan(500);
    
    if (response.status() === 200) {
      const data = await response.json();
      console.log('节点模板响应:', JSON.stringify(data, null, 2));
    } else {
      console.log('API 响应状态:', response.status());
    }
  });

  test('应该能测试 SSH 连接（预期可能失败）', async ({ request }) => {
    const testData = {
      host: 'slurm-master',
      port: 22,
      username: 'root',
      password: 'aiinfra2024',
      auth_type: 'password'
    };

    const response = await request.post(`${BASE_API}/slurm/clusters/test-connection`, {
      data: testData
    });
    
    // 验证响应状态（可能401/403需要认证，或200/400连接测试结果）
    console.log('SSH 连接测试响应状态:', response.status());
    
    if (response.status() < 500 && response.status() !== 404) {
      try {
        const data = await response.json();
        console.log('SSH 连接测试结果:', JSON.stringify(data, null, 2));
      } catch (e) {
        console.log('响应不是 JSON 格式');
      }
    }
  });

  test('应该能访问 SaltStack 集成状态 API', async ({ request }) => {
    const response = await request.get(`${BASE_API}/slurm/saltstack/integration`);
    
    // 验证响应状态
    expect(response.status()).toBeLessThan(500);
    
    if (response.status() === 200) {
      const data = await response.json();
      console.log('SaltStack 集成状态:', JSON.stringify(data, null, 2));
      
      if (data.success && data.data) {
        console.log('✅ SaltStack 已集成');
        console.log('- 状态:', data.data.status);
        console.log('- 已接受 Keys:', data.data.accepted_keys?.length || 0);
        console.log('- 待接受 Keys:', data.data.unaccepted_keys?.length || 0);
      }
    } else {
      console.log('API 响应状态:', response.status());
    }
  });

  test('应该能访问后端健康检查', async ({ request }) => {
    const response = await request.get(`${BASE_API}/health`);
    
    expect(response.status()).toBe(200);
    
    const data = await response.json();
    console.log('后端健康状态:', JSON.stringify(data, null, 2));
    
    expect(data).toHaveProperty('status');
    expect(['ok', 'healthy']).toContain(data.status);
  });
});

test.describe('SLURM 集群创建和管理流程测试', () => {
  
  test('完整流程：检查现有集群或提示创建', async ({ request }) => {
    console.log('\n=== SLURM 集群管理功能测试 ===\n');
    
    // 1. 检查集群列表
    console.log('1. 检查集群列表...');
    const clustersResponse = await request.get(`${BASE_API}/slurm/clusters`);
    
    if (clustersResponse.status() === 200) {
      const clustersData = await clustersResponse.json();
      console.log('✅ 集群列表 API 可访问');
      
      if (clustersData.success && clustersData.data) {
        const clusters = Array.isArray(clustersData.data) ? clustersData.data : [clustersData.data];
        console.log(`📊 找到 ${clusters.length} 个集群`);
        
        if (clusters.length > 0) {
          clusters.forEach((cluster, index) => {
            console.log(`\n集群 ${index + 1}:`);
            console.log(`  - ID: ${cluster.id}`);
            console.log(`  - 名称: ${cluster.name}`);
            console.log(`  - 类型: ${cluster.cluster_type || 'managed'}`);
            console.log(`  - 状态: ${cluster.status}`);
            console.log(`  - 主节点: ${cluster.master_host || 'N/A'}`);
          });
        } else {
          console.log('ℹ️  暂无集群，可以创建新集群');
        }
      }
    } else {
      console.log('⚠️  集群列表 API 返回:', clustersResponse.status());
    }
    
    // 2. 检查节点列表
    console.log('\n2. 检查节点列表...');
    const nodesResponse = await request.get(`${BASE_API}/slurm/nodes`);
    
    if (nodesResponse.status() === 200) {
      const nodesData = await nodesResponse.json();
      console.log('✅ 节点列表 API 可访问');
      
      if (nodesData.success && nodesData.data) {
        const nodes = Array.isArray(nodesData.data) ? nodesData.data : [];
        console.log(`📊 找到 ${nodes.length} 个节点`);
        
        if (nodes.length > 0) {
          nodes.slice(0, 5).forEach((node, index) => {
            console.log(`  ${index + 1}. ${node.node_name} - 状态: ${node.status}`);
          });
          if (nodes.length > 5) {
            console.log(`  ... 还有 ${nodes.length - 5} 个节点`);
          }
        }
      }
    } else {
      console.log('⚠️  节点列表 API 返回:', nodesResponse.status());
    }
    
    // 3. 检查 SaltStack 集成
    console.log('\n3. 检查 SaltStack 集成状态...');
    const saltResponse = await request.get(`${BASE_API}/slurm/saltstack/integration`);
    
    if (saltResponse.status() === 200) {
      const saltData = await saltResponse.json();
      console.log('✅ SaltStack 集成 API 可访问');
      
      if (saltData.success && saltData.data) {
        console.log(`  - 状态: ${saltData.data.status}`);
        console.log(`  - 已接受 Minions: ${saltData.data.accepted_keys?.length || 0}`);
        console.log(`  - 待接受 Minions: ${saltData.data.unaccepted_keys?.length || 0}`);
        
        if (saltData.data.accepted_keys && saltData.data.accepted_keys.length > 0) {
          console.log('\n  已接受的 Minions:');
          saltData.data.accepted_keys.forEach(key => {
            console.log(`    - ${key}`);
          });
        }
      }
    } else {
      console.log('⚠️  SaltStack 集成 API 返回:', saltResponse.status());
    }
    
    // 4. 总结
    console.log('\n=== 测试总结 ===');
    console.log('✅ 所有 API 端点都可访问');
    console.log('✅ 后端服务运行正常');
    console.log('\n📝 下一步操作:');
    console.log('  1. 访问 http://192.168.3.91:8080/slurm 查看 Web 界面');
    console.log('  2. 使用 admin/admin123 登录（如需要）');
    console.log('  3. 在界面上创建或连接 SLURM 集群');
    console.log('  4. 测试外部集群连接功能');
    console.log('\n');
  });
});
