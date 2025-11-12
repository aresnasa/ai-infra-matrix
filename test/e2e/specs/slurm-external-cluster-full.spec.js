const { test, expect } = require('@playwright/test');

/**
 * SLURM 外部集群管理完整测试
 * 包含登录、访问页面、测试 API
 */

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.91:8080';
const API_URL = `${BASE_URL}/api`;

// 测试用户凭证
const TEST_USER = {
  username: 'admin',
  password: 'admin123'
};

test.describe('SLURM 外部集群管理 - 完整流程', () => {
  
  test('完整流程: 登录 → 访问页面 → 测试外部集群管理', async ({ page }) => {
    console.log('========================================');
    console.log('步骤 1: 访问登录页面');
    console.log('========================================');
    
    await page.goto(`${BASE_URL}/login`);
    await page.waitForLoadState('networkidle');
    
    // 检查是否已经登录
    const currentUrl = page.url();
    if (!currentUrl.includes('/login')) {
      console.log('✓ 已经登录，跳过登录步骤');
    } else {
      console.log('执行登录...');
      
      // 填写登录表单
      await page.fill('input[type="text"], input[name="username"]', TEST_USER.username);
      await page.fill('input[type="password"], input[name="password"]', TEST_USER.password);
      
      // 点击登录按钮
      await page.click('button[type="submit"], button:has-text("登录")');
      
      // 等待登录完成
      await page.waitForURL(/\/(dashboard|home|slurm|projects)/, { timeout: 10000 });
      console.log('✓ 登录成功');
    }
    
    console.log('========================================');
    console.log('步骤 2: 访问 SLURM 管理页面');
    console.log('========================================');
    
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    
    // 验证页面加载
    const title = await page.title();
    console.log('页面标题:', title);
    
    // 检查是否有 SLURM 相关元素
    const hasContent = await page.locator('text=/SLURM|集群|节点/i').first().isVisible({ timeout: 5000 }).catch(() => false);
    expect(hasContent).toBeTruthy();
    console.log('✓ SLURM 页面加载成功');
    
    console.log('========================================');
    console.log('步骤 3: 查找并点击"外部集群管理"标签');
    console.log('========================================');
    
    // 等待标签页加载
    await page.waitForTimeout(1000);
    
    // 查找"外部集群管理"标签
    const externalClusterTab = page.locator('text=/外部集群管理/i');
    const tabExists = await externalClusterTab.count();
    
    if (tabExists > 0) {
      console.log('✓ 找到"外部集群管理"标签');
      await externalClusterTab.first().click();
      await page.waitForTimeout(1000);
      console.log('✓ 已切换到"外部集群管理"标签');
      
      // 验证标签内容加载
      const hasTable = await page.locator('table, .ant-table').isVisible({ timeout: 3000 }).catch(() => false);
      const hasCard = await page.locator('.ant-card').isVisible({ timeout: 3000 }).catch(() => false);
      
      if (hasTable || hasCard) {
        console.log('✓ 外部集群管理界面加载成功');
      } else {
        console.log('⚠ 界面元素未完全加载');
      }
    } else {
      console.log('⚠ 未找到"外部集群管理"标签，可能未集成到页面');
    }
    
    console.log('========================================');
    console.log('步骤 4: 测试 API 调用');
    console.log('========================================');
    
    // 获取 Cookie 用于 API 请求
    const cookies = await page.context().cookies();
    const cookieHeader = cookies.map(c => `${c.name}=${c.value}`).join('; ');
    
    console.log('4.1 测试获取集群列表 API');
    
    // 先检查 cookies
    const cookiesBeforeAPI = await page.context().cookies();
    console.log('页面中的 Cookies:');
    cookiesBeforeAPI.forEach(c => {
      console.log(`  - ${c.name}: ${c.value.substring(0, 30)}... (domain: ${c.domain}, path: ${c.path})`);
    });
    
    const listResponse = await page.evaluate(async (apiUrl) => {
      try {
        const res = await fetch(`${apiUrl}/slurm/clusters`, {
          method: 'GET',
          credentials: 'include'
        });
        return {
          status: res.status,
          statusText: res.statusText,
          data: await res.json().catch(() => ({}))
        };
      } catch (error) {
        return { error: error.message };
      }
    }, API_URL);
    
    console.log('API 响应:', JSON.stringify(listResponse, null, 2));
    
    if (listResponse.status === 200) {
      console.log('✓ 集群列表 API 调用成功');
    } else if (listResponse.status === 401) {
      console.log('✗ 认证失败，需要检查后端认证逻辑');
      expect(listResponse.status).not.toBe(401);
    } else {
      console.log(`⚠ API 返回状态码: ${listResponse.status}`);
    }
    
    console.log('========================================');
    console.log('步骤 5: 检查页面错误信息');
    console.log('========================================');
    
    // 检查是否有错误提示
    const errorMessages = await page.locator('.ant-message-error, .ant-alert-error').allTextContents();
    if (errorMessages.length > 0) {
      console.log('页面错误信息:', errorMessages);
    } else {
      console.log('✓ 没有发现错误信息');
    }
    
    // 截图保存
    await page.screenshot({ 
      path: 'test-screenshots/slurm-external-cluster-full.png',
      fullPage: true 
    });
    console.log('✓ 已保存截图: test-screenshots/slurm-external-cluster-full.png');
    
    console.log('========================================');
    console.log('测试完成');
    console.log('========================================');
  });
  
  test('仅测试 API - 获取集群列表（需要认证）', async ({ request, page }) => {
    console.log('登录以获取认证 Cookie...');
    
    // 先访问登录页面
    await page.goto(`${BASE_URL}/login`);
    
    // 检查是否需要登录
    if (page.url().includes('/login')) {
      await page.fill('input[type="text"], input[name="username"]', TEST_USER.username);
      await page.fill('input[type="password"], input[name="password"]', TEST_USER.password);
      await page.click('button[type="submit"], button:has-text("登录")');
      await page.waitForURL(/\/(dashboard|home|slurm|projects)/, { timeout: 10000 });
    }
    
    // 获取认证 Cookie
    const cookies = await page.context().cookies();
    console.log('所有 Cookies:', cookies.map(c => `${c.name}=${c.value.substring(0, 20)}...`));
    const authToken = cookies.find(c => c.name === 'auth_token');
    
    if (!authToken) {
      console.log('⚠ 未找到 auth_token cookie，可用的 cookies:', cookies.map(c => c.name).join(', '));
    }
    
    const cookieHeader = cookies.map(c => `${c.name}=${c.value}`).join('; ');
    
    console.log('测试集群列表 API...');
    const response = await request.get(`${API_URL}/slurm/clusters`, {
      headers: {
        'Cookie': cookieHeader
      }
    });
    
    console.log('响应状态:', response.status());
    
    if (response.status() === 200) {
      const data = await response.json();
      console.log('集群数据:', JSON.stringify(data, null, 2));
      expect(data).toBeDefined();
    } else if (response.status() === 401) {
      console.log('✗ 认证失败');
      const body = await response.text();
      console.log('响应内容:', body);
      expect(response.status()).not.toBe(401);
    } else {
      console.log('⚠ 返回状态码:', response.status());
      const body = await response.text();
      console.log('响应内容:', body);
    }
  });
  
  test('检查后端健康状态', async ({ request }) => {
    console.log('检查后端服务...');
    
    const healthResponse = await request.get(`${API_URL}/health`).catch(() => null);
    
    if (healthResponse) {
      console.log('后端健康检查状态:', healthResponse.status());
      if (healthResponse.ok()) {
        const data = await healthResponse.json();
        console.log('健康检查数据:', data);
      }
    } else {
      console.log('⚠ 后端服务可能未启动');
    }
  });
});
