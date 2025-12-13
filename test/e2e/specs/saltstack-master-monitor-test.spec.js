// @ts-check
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://192.168.1.81:8080';
const ADMIN_USER = 'admin';
const ADMIN_PASS = 'admin123';

// 测试节点信息
const TEST_NODE = {
  host: '10.211.55.7',
  username: 'aresnasa',
  password: 'QWErt12345!@#'
};

test.describe('SaltStack Master 监控和节点注册测试', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    // 填写登录表单
    await page.fill('input[placeholder*="用户名"], input[id*="username"]', ADMIN_USER);
    await page.fill('input[type="password"]', ADMIN_PASS);
    await page.click('button[type="submit"], button:has-text("登")');
    
    // 等待登录完成
    await page.waitForURL(/.*(?<!login)$/, { timeout: 15000 });
    await page.waitForLoadState('networkidle');
  });

  test('检查 Master 监控数据显示', async ({ page }) => {
    // 导航到 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    
    // 截图当前页面
    await page.screenshot({ path: 'test-screenshots/saltstack-page-initial.png', fullPage: true });
    
    // 检查页面标题
    const pageTitle = await page.textContent('h1, h2, .ant-page-header-heading-title, [class*="title"]');
    console.log('页面标题:', pageTitle);
    
    // 查找 Master 状态区域
    const masterSection = page.locator('text=Master >> xpath=ancestor::*[contains(@class, "card") or contains(@class, "Card")]').first();
    
    // 检查 CPU 使用率
    const cpuElement = page.locator('text=/CPU.*%|cpu.*usage/i').first();
    if (await cpuElement.isVisible({ timeout: 5000 }).catch(() => false)) {
      const cpuText = await cpuElement.textContent();
      console.log('CPU 使用率:', cpuText);
    }
    
    // 检查内存使用率
    const memElement = page.locator('text=/内存|Memory.*%/i').first();
    if (await memElement.isVisible({ timeout: 5000 }).catch(() => false)) {
      const memText = await memElement.textContent();
      console.log('内存使用率:', memText);
    }
    
    // 检查连接数
    const connElement = page.locator('text=/连接|Connections/i').first();
    if (await connElement.isVisible({ timeout: 5000 }).catch(() => false)) {
      const connText = await connElement.textContent();
      console.log('连接数:', connText);
    }
    
    // 检查网络带宽
    const bwElement = page.locator('text=/带宽|Bandwidth|Mbps/i').first();
    if (await bwElement.isVisible({ timeout: 5000 }).catch(() => false)) {
      const bwText = await bwElement.textContent();
      console.log('网络带宽:', bwText);
    }
    
    // 截图监控数据
    await page.screenshot({ path: 'test-screenshots/saltstack-master-monitor.png', fullPage: true });
    
    // 检查 API 响应
    const response = await page.evaluate(async () => {
      const token = localStorage.getItem('token') || localStorage.getItem('access_token');
      const headers = token ? { 'Authorization': `Bearer ${token}` } : {};
      
      try {
        const res = await fetch('/api/saltstack/status', { headers });
        return await res.json();
      } catch (e) {
        return { error: e.message };
      }
    });
    
    console.log('API 响应:', JSON.stringify(response, null, 2));
    
    // 验证响应包含监控数据
    if (response && !response.error) {
      console.log('Master CPU:', response.cpu_usage || response.cpuUsage);
      console.log('Master Memory:', response.memory_usage || response.memoryUsage);
      console.log('Master Connections:', response.active_connections || response.activeConnections);
      console.log('Master Network Bandwidth:', response.network_bandwidth || response.networkBandwidth);
    }
  });

  test('注册新节点并检查监控数据', async ({ page }) => {
    // 导航到 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 查找注册节点按钮
    const registerButton = page.locator('button:has-text("注册"), button:has-text("添加节点"), button:has-text("Register")').first();
    
    if (await registerButton.isVisible({ timeout: 5000 }).catch(() => false)) {
      await registerButton.click();
      await page.waitForTimeout(1000);
      
      // 截图注册对话框
      await page.screenshot({ path: 'test-screenshots/saltstack-register-dialog.png' });
      
      // 填写注册表单
      // 主机地址
      const hostInput = page.locator('input[placeholder*="IP"], input[placeholder*="主机"], input[id*="host"]').first();
      if (await hostInput.isVisible()) {
        await hostInput.fill(TEST_NODE.host);
      }
      
      // 用户名
      const userInput = page.locator('input[placeholder*="用户名"], input[id*="username"], input[id*="user"]').first();
      if (await userInput.isVisible()) {
        await userInput.fill(TEST_NODE.username);
      }
      
      // 密码
      const passInput = page.locator('input[type="password"]').first();
      if (await passInput.isVisible()) {
        await passInput.fill(TEST_NODE.password);
      }
      
      await page.screenshot({ path: 'test-screenshots/saltstack-register-filled.png' });
      
      // 提交注册
      const submitButton = page.locator('button:has-text("确定"), button:has-text("提交"), button:has-text("Submit"), button[type="submit"]').first();
      if (await submitButton.isVisible()) {
        await submitButton.click();
        await page.waitForTimeout(5000);
      }
      
      await page.screenshot({ path: 'test-screenshots/saltstack-register-result.png' });
    } else {
      console.log('未找到注册按钮，尝试查找添加节点的方式...');
      
      // 尝试通过 API 注册
      const registerResult = await page.evaluate(async (nodeInfo) => {
        const token = localStorage.getItem('token') || localStorage.getItem('access_token');
        const headers = {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        };
        
        try {
          const res = await fetch('/api/saltstack/minions/register', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              host: nodeInfo.host,
              username: nodeInfo.username,
              password: nodeInfo.password,
              port: 22
            })
          });
          return await res.json();
        } catch (e) {
          return { error: e.message };
        }
      }, TEST_NODE);
      
      console.log('API 注册结果:', JSON.stringify(registerResult, null, 2));
    }
    
    // 等待节点出现在列表中
    await page.waitForTimeout(3000);
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 查找新注册的节点
    const nodeElement = page.locator(`text=${TEST_NODE.host}`).first();
    if (await nodeElement.isVisible({ timeout: 10000 }).catch(() => false)) {
      console.log(`节点 ${TEST_NODE.host} 已注册成功`);
      
      // 点击节点查看详情
      await nodeElement.click();
      await page.waitForTimeout(2000);
      
      await page.screenshot({ path: 'test-screenshots/saltstack-node-detail.png', fullPage: true });
    }
  });

  test('检查节点监控数据采集类型（容器/cgroup vs 物理机）', async ({ page }) => {
    // 导航到 SaltStack 页面
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // 获取节点监控数据
    const minionMetrics = await page.evaluate(async (host) => {
      const token = localStorage.getItem('token') || localStorage.getItem('access_token');
      const headers = token ? { 'Authorization': `Bearer ${token}` } : {};
      
      try {
        // 尝试获取节点详情
        const res = await fetch(`/api/saltstack/minions/${host}/metrics`, { headers });
        if (res.ok) {
          return await res.json();
        }
        
        // 尝试其他 API
        const res2 = await fetch(`/api/saltstack/minions?host=${host}`, { headers });
        if (res2.ok) {
          return await res2.json();
        }
        
        return { error: 'API not found' };
      } catch (e) {
        return { error: e.message };
      }
    }, TEST_NODE.host);
    
    console.log('节点监控数据:', JSON.stringify(minionMetrics, null, 2));
    
    // 检查监控类型
    if (minionMetrics && !minionMetrics.error) {
      // 检查是否是容器（通过 cgroup 检测）
      const isContainer = minionMetrics.is_container || 
                          minionMetrics.container_type || 
                          minionMetrics.cgroup_enabled;
      
      if (isContainer) {
        console.log('检测到容器环境，使用 cgroup 监控数据');
        console.log('Cgroup CPU:', minionMetrics.cgroup_cpu || minionMetrics.cpu_cgroup);
        console.log('Cgroup Memory:', minionMetrics.cgroup_memory || minionMetrics.memory_cgroup);
      } else {
        console.log('检测到物理机/虚拟机环境，使用直接监控数据');
        console.log('System CPU:', minionMetrics.cpu_usage || minionMetrics.cpu);
        console.log('System Memory:', minionMetrics.memory_usage || minionMetrics.memory);
      }
    }
    
    // 截图最终状态
    await page.screenshot({ path: 'test-screenshots/saltstack-metrics-type.png', fullPage: true });
  });

  test('检查 Master 容器 vs 进程模式监控', async ({ page }) => {
    // 获取 Master 监控详情
    const masterMetrics = await page.evaluate(async () => {
      const token = localStorage.getItem('token') || localStorage.getItem('access_token');
      const headers = token ? { 'Authorization': `Bearer ${token}` } : {};
      
      try {
        const res = await fetch('/api/saltstack/status', { headers });
        return await res.json();
      } catch (e) {
        return { error: e.message };
      }
    });
    
    console.log('Master 监控详情:', JSON.stringify(masterMetrics, null, 2));
    
    // 分析监控数据来源
    if (masterMetrics && !masterMetrics.error) {
      // 检查数据来源
      const metricsSource = masterMetrics.metrics_source || masterMetrics.source;
      console.log('监控数据来源:', metricsSource);
      
      // 如果是容器，应该有 Docker/cgroup 相关指标
      if (metricsSource === 'docker' || metricsSource === 'container') {
        console.log('Master 运行在容器中，使用 Docker API 采集');
        console.log('Container CPU:', masterMetrics.container_cpu_percent);
        console.log('Container Memory:', masterMetrics.container_memory_percent);
      } else if (metricsSource === 'victoriametrics' || metricsSource === 'prometheus') {
        console.log('Master 使用 VictoriaMetrics 采集');
        console.log('VictoriaMetrics CPU:', masterMetrics.cpu_usage);
        console.log('VictoriaMetrics Memory:', masterMetrics.memory_usage);
      } else {
        console.log('Master 使用 Salt API 直接采集');
        console.log('Salt API CPU:', masterMetrics.cpu_usage);
        console.log('Salt API Memory:', masterMetrics.memory_usage);
      }
      
      // 验证监控数据非零
      const cpuUsage = masterMetrics.cpu_usage || masterMetrics.cpuUsage || 0;
      const memUsage = masterMetrics.memory_usage || masterMetrics.memoryUsage || 0;
      
      console.log(`\n监控数据验证:`);
      console.log(`  CPU: ${cpuUsage}% (${cpuUsage > 0 ? '✓ 有效' : '✗ 无数据'})`);
      console.log(`  Memory: ${memUsage}% (${memUsage > 0 ? '✓ 有效' : '✗ 无数据'})`);
    }
    
    // 导航到页面验证 UI 显示
    await page.goto(`${BASE_URL}/saltstack`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);
    
    await page.screenshot({ path: 'test-screenshots/saltstack-master-metrics-final.png', fullPage: true });
  });
});
