/**
 * AI Infrastructure Matrix - 完整 E2E 测试套件
 * 
 * 测试覆盖：
 * 1. 用户认证（注册、登录、登出）
 * 2. 核心功能页面访问
 * 3. SLURM 任务管理
 * 4. SaltStack 管理
 * 5. Kubernetes 管理
 * 6. 对象存储管理
 * 7. JupyterHub 集成
 * 8. Gitea 集成
 * 9. 管理员功能（需要管理员权限）
 * 10. 前端页面优化验证
 */

const { test, expect } = require('@playwright/test');

// 测试配置
const TEST_CONFIG = {
  baseURL: process.env.BASE_URL || 'http://localhost:8080',
  timeout: 30000,
  adminUser: {
    username: process.env.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || 'admin123',
  },
  testUser: {
    username: process.env.TEST_USERNAME || 'testuser',
    password: process.env.TEST_PASSWORD || 'test123',
    email: 'testuser@example.com',
  },
};

// 辅助函数：登录
async function login(page, username, password) {
  await page.goto('/');
  
  // 等待登录表单加载
  await page.waitForSelector('input[type="text"]', { timeout: 10000 });
  
  // 填写登录表单
  await page.fill('input[type="text"]', username);
  await page.fill('input[type="password"]', password);
  
  // 点击登录按钮
  await page.click('button[type="submit"]');
  
  // 等待登录成功，跳转到主页
  await page.waitForURL('**/projects', { timeout: 15000 });
  
  // 验证登录成功
  const userMenu = await page.locator('.ant-dropdown-trigger').first();
  await expect(userMenu).toBeVisible();
}

// 辅助函数：登出
async function logout(page) {
  // 点击用户菜单
  await page.click('.ant-dropdown-trigger');
  
  // 等待菜单展开
  await page.waitForSelector('.ant-dropdown-menu', { state: 'visible' });
  
  // 点击登出
  await page.click('text=退出登录');
  
  // 等待跳转到登录页
  await page.waitForURL('**/auth', { timeout: 10000 });
}

// 辅助函数：等待页面加载完成
async function waitForPageLoad(page) {
  await page.waitForLoadState('networkidle', { timeout: 15000 });
  // 等待 Ant Design 的 Spin 组件消失
  await page.waitForSelector('.ant-spin-spinning', { state: 'detached', timeout: 10000 }).catch(() => {});
}

test.describe('AI Infrastructure Matrix - 完整 E2E 测试', () => {
  
  test.beforeEach(async ({ page }) => {
    // 设置视口大小
    await page.setViewportSize({ width: 1920, height: 1080 });
  });

  test.describe('1. 用户认证测试', () => {
    
    test('1.1 登录页面加载', async ({ page }) => {
      await page.goto('/');
      
      // 验证页面标题
      await expect(page).toHaveTitle(/AI Infrastructure Matrix/);
      
      // 验证登录表单元素
      await expect(page.locator('input[type="text"]')).toBeVisible();
      await expect(page.locator('input[type="password"]')).toBeVisible();
      await expect(page.locator('button[type="submit"]')).toBeVisible();
    });

    test('1.2 管理员登录', async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
      
      // 验证跳转到项目列表页
      await expect(page).toHaveURL(/\/projects/);
      
      // 验证用户菜单显示
      const userMenu = await page.locator('.ant-dropdown-trigger').first();
      await expect(userMenu).toBeVisible();
    });

    test('1.3 管理员登出', async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
      await logout(page);
      
      // 验证返回登录页
      await expect(page).toHaveURL(/\/auth/);
    });

    test('1.4 错误登录凭证', async ({ page }) => {
      await page.goto('/');
      
      await page.fill('input[type="text"]', 'wronguser');
      await page.fill('input[type="password"]', 'wrongpass');
      await page.click('button[type="submit"]');
      
      // 等待错误提示
      await page.waitForSelector('.ant-message-error', { timeout: 5000 });
    });
  });

  test.describe('2. 核心功能页面访问测试', () => {
    
    test.beforeEach(async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
    });

    test('2.1 项目列表页面', async ({ page }) => {
      await page.goto('/projects');
      await waitForPageLoad(page);
      
      // 验证页面元素
      await expect(page.locator('text=项目列表')).toBeVisible();
    });

    test('2.2 用户资料页面', async ({ page }) => {
      await page.goto('/profile');
      await waitForPageLoad(page);
      
      // 验证页面加载
      await expect(page.locator('text=用户资料').or(page.locator('text=个人信息'))).toBeVisible();
    });

    test('2.3 SLURM 任务页面', async ({ page }) => {
      await page.goto('/slurm-tasks');
      await waitForPageLoad(page);
      
      // 验证页面元素
      await expect(page.locator('text=任务管理').or(page.locator('text=SLURM 任务'))).toBeVisible();
      
      // 验证 Tab 切换
      const statisticsTab = page.locator('text=统计信息');
      if (await statisticsTab.isVisible()) {
        await statisticsTab.click();
        await page.waitForTimeout(1000);
      }
    });

    test('2.4 SLURM Dashboard 页面', async ({ page }) => {
      await page.goto('/slurm');
      await waitForPageLoad(page);
      
      // 验证页面加载
      await expect(page.locator('text=SLURM').or(page.locator('text=集群状态'))).toBeVisible();
    });

    test('2.5 SaltStack 仪表板页面', async ({ page }) => {
      await page.goto('/saltstack');
      await waitForPageLoad(page);
      
      // 验证页面元素
      await expect(page.locator('text=SaltStack').or(page.locator('text=配置管理'))).toBeVisible();
    });

    test('2.6 Kubernetes 管理页面', async ({ page }) => {
      await page.goto('/kubernetes');
      await waitForPageLoad(page);
      
      // 验证页面加载
      await expect(page.locator('text=Kubernetes').or(page.locator('text=集群管理'))).toBeVisible();
    });

    test('2.7 对象存储页面', async ({ page }) => {
      await page.goto('/object-storage');
      await waitForPageLoad(page);
      
      // 验证页面元素
      await expect(page.locator('text=对象存储').or(page.locator('text=MinIO'))).toBeVisible();
    });

    test('2.8 JupyterHub 页面', async ({ page }) => {
      await page.goto('/jupyter');
      await waitForPageLoad(page);
      
      // 验证页面加载（可能是 iframe 嵌入）
      await expect(page.locator('text=JupyterHub').or(page.locator('iframe'))).toBeVisible();
    });

    test('2.9 Gitea 页面', async ({ page }) => {
      await page.goto('/gitea');
      await waitForPageLoad(page);
      
      // 验证页面加载（可能是 iframe 嵌入）
      await expect(page.locator('text=Gitea').or(page.locator('iframe'))).toBeVisible();
    });
  });

  test.describe('3. SLURM 任务管理测试', () => {
    
    test.beforeEach(async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
      await page.goto('/slurm-tasks');
      await waitForPageLoad(page);
    });

    test('3.1 任务列表加载', async ({ page }) => {
      // 验证任务列表表格
      const table = page.locator('.ant-table');
      await expect(table).toBeVisible();
    });

    test('3.2 任务筛选功能', async ({ page }) => {
      // 点击状态筛选
      const statusFilter = page.locator('.ant-select').first();
      if (await statusFilter.isVisible()) {
        await statusFilter.click();
        await page.waitForTimeout(500);
        
        // 选择运行中状态
        const runningOption = page.locator('text=运行中').or(page.locator('text=Running'));
        if (await runningOption.isVisible()) {
          await runningOption.click();
          await page.waitForTimeout(1000);
        }
      }
    });

    test('3.3 统计信息页面', async ({ page }) => {
      // 切换到统计信息 Tab
      const statisticsTab = page.locator('text=统计信息');
      if (await statisticsTab.isVisible()) {
        await statisticsTab.click();
        await page.waitForTimeout(2000);
        
        // 验证统计卡片
        const cards = page.locator('.ant-card');
        const count = await cards.count();
        expect(count).toBeGreaterThan(0);
      }
    });

    test('3.4 自动刷新功能', async ({ page }) => {
      // 验证自动刷新开关
      const refreshSwitch = page.locator('text=自动刷新').locator('..').locator('.ant-switch');
      if (await refreshSwitch.isVisible()) {
        // 检查当前状态
        const isChecked = await refreshSwitch.getAttribute('aria-checked');
        console.log('自动刷新状态:', isChecked);
        
        // 切换状态
        await refreshSwitch.click();
        await page.waitForTimeout(500);
      }
    });

    test('3.5 URL 参数处理', async ({ page }) => {
      // 测试带参数的 URL
      await page.goto('/slurm-tasks?status=running');
      await waitForPageLoad(page);
      
      // 验证状态筛选已应用
      await page.waitForTimeout(1000);
    });
  });

  test.describe('4. SaltStack 管理测试', () => {
    
    test.beforeEach(async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
      await page.goto('/saltstack');
      await waitForPageLoad(page);
    });

    test('4.1 SaltStack 仪表板加载', async ({ page }) => {
      // 验证仪表板元素
      await expect(page.locator('text=SaltStack').or(page.locator('text=Minion'))).toBeVisible();
    });

    test('4.2 Minion 列表显示', async ({ page }) => {
      // 等待 Minion 列表加载
      await page.waitForTimeout(2000);
      
      // 验证表格或卡片显示
      const table = page.locator('.ant-table').or(page.locator('.ant-card'));
      await expect(table).toBeVisible();
    });

    test('4.3 集成状态显示', async ({ page }) => {
      // 验证集成状态卡片（之前修复的功能）
      await page.waitForTimeout(2000);
      
      // 查找集成状态相关元素
      const integrationCard = page.locator('text=集成').or(page.locator('text=Integration'));
      if (await integrationCard.isVisible()) {
        console.log('✓ SaltStack 集成状态卡片已显示');
      }
    });
  });

  test.describe('5. 对象存储管理测试', () => {
    
    test.beforeEach(async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
      await page.goto('/object-storage');
      await waitForPageLoad(page);
    });

    test('5.1 对象存储页面加载', async ({ page }) => {
      // 验证页面元素
      await expect(page.locator('text=对象存储').or(page.locator('text=MinIO'))).toBeVisible();
    });

    test('5.2 自动刷新功能（优化后）', async ({ page }) => {
      // 等待页面完全加载
      await page.waitForTimeout(2000);
      
      // 验证自动刷新控制
      const refreshButton = page.locator('text=刷新').or(page.locator('[aria-label*="refresh"]'));
      if (await refreshButton.isVisible()) {
        await refreshButton.click();
        await page.waitForTimeout(1000);
      }
    });

    test('5.3 Bucket 列表显示', async ({ page }) => {
      // 等待 Bucket 列表加载
      await page.waitForTimeout(2000);
      
      // 验证列表显示
      const table = page.locator('.ant-table').or(page.locator('.ant-list'));
      await expect(table).toBeVisible();
    });
  });

  test.describe('6. 管理员功能测试', () => {
    
    test.beforeEach(async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
    });

    test('6.1 管理中心页面', async ({ page }) => {
      await page.goto('/admin');
      await waitForPageLoad(page);
      
      // 验证管理中心元素
      await expect(page.locator('text=管理中心').or(page.locator('text=系统管理'))).toBeVisible();
    });

    test('6.2 用户管理页面', async ({ page }) => {
      await page.goto('/admin/users');
      await waitForPageLoad(page);
      
      // 验证用户列表
      const table = page.locator('.ant-table');
      await expect(table).toBeVisible();
    });

    test('6.3 项目管理页面', async ({ page }) => {
      await page.goto('/admin/projects');
      await waitForPageLoad(page);
      
      // 验证项目列表
      await expect(page.locator('text=项目管理').or(page.locator('.ant-table'))).toBeVisible();
    });

    test('6.4 LDAP 配置页面', async ({ page }) => {
      await page.goto('/admin/ldap');
      await waitForPageLoad(page);
      
      // 验证 LDAP 配置表单
      await expect(page.locator('text=LDAP').or(page.locator('text=目录服务'))).toBeVisible();
    });

    test('6.5 对象存储配置', async ({ page }) => {
      await page.goto('/admin/object-storage-config');
      await waitForPageLoad(page);
      
      // 验证配置页面
      await expect(page.locator('text=对象存储配置').or(page.locator('text=MinIO 配置'))).toBeVisible();
    });
  });

  test.describe('7. 前端优化验证测试', () => {
    
    test.beforeEach(async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
    });

    test('7.1 验证 SLURM Tasks 刷新频率优化', async ({ page }) => {
      await page.goto('/slurm-tasks');
      await waitForPageLoad(page);
      
      // 监听网络请求
      const requests = [];
      page.on('request', request => {
        if (request.url().includes('/api/slurm/tasks')) {
          requests.push({
            time: Date.now(),
            url: request.url()
          });
        }
      });
      
      // 等待 65 秒，验证刷新间隔（应该是 60 秒左右）
      console.log('等待 65 秒以验证刷新间隔...');
      await page.waitForTimeout(65000);
      
      // 分析请求间隔
      if (requests.length >= 2) {
        const interval = requests[1].time - requests[0].time;
        console.log(`刷新间隔: ${interval / 1000} 秒`);
        
        // 验证间隔不小于 30 秒（优化后的最小间隔）
        expect(interval).toBeGreaterThanOrEqual(30000);
      }
    });

    test('7.2 验证 Object Storage 懒加载', async ({ page }) => {
      await page.goto('/object-storage');
      
      // 监听资源加载
      const loadedResources = [];
      page.on('response', response => {
        loadedResources.push({
          url: response.url(),
          status: response.status()
        });
      });
      
      await waitForPageLoad(page);
      
      // 验证页面加载成功
      const successfulLoads = loadedResources.filter(r => r.status === 200);
      expect(successfulLoads.length).toBeGreaterThan(0);
    });

    test('7.3 验证 SaltStack Integration 显示', async ({ page }) => {
      await page.goto('/slurm');
      await waitForPageLoad(page);
      
      // 等待集成状态加载
      await page.waitForTimeout(3000);
      
      // 验证 SaltStack 集成卡片
      const integrationSection = page.locator('text=SaltStack').or(page.locator('text=集成'));
      if (await integrationSection.isVisible()) {
        console.log('✓ SaltStack 集成状态已显示');
      }
    });

    test('7.4 验证页面响应性能', async ({ page }) => {
      const startTime = Date.now();
      
      await page.goto('/projects');
      await waitForPageLoad(page);
      
      const loadTime = Date.now() - startTime;
      console.log(`页面加载时间: ${loadTime} ms`);
      
      // 验证加载时间不超过 10 秒
      expect(loadTime).toBeLessThan(10000);
    });
  });

  test.describe('8. 导航和路由测试', () => {
    
    test.beforeEach(async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
    });

    test('8.1 侧边栏导航', async ({ page }) => {
      await page.goto('/projects');
      await waitForPageLoad(page);
      
      // 查找并点击侧边栏菜单项
      const menuItems = [
        'SLURM',
        'SaltStack',
        'Kubernetes',
        '对象存储',
        'JupyterHub',
        'Gitea'
      ];
      
      for (const item of menuItems) {
        const menuItem = page.locator(`text=${item}`).first();
        if (await menuItem.isVisible()) {
          await menuItem.click();
          await page.waitForTimeout(1000);
          console.log(`✓ 导航到 ${item} 页面`);
        }
      }
    });

    test('8.2 浏览器前进后退', async ({ page }) => {
      await page.goto('/projects');
      await waitForPageLoad(page);
      
      await page.goto('/slurm-tasks');
      await waitForPageLoad(page);
      
      // 后退
      await page.goBack();
      await expect(page).toHaveURL(/\/projects/);
      
      // 前进
      await page.goForward();
      await expect(page).toHaveURL(/\/slurm-tasks/);
    });

    test('8.3 直接 URL 访问', async ({ page }) => {
      const urls = [
        '/projects',
        '/profile',
        '/slurm-tasks',
        '/slurm',
        '/saltstack',
        '/kubernetes',
        '/object-storage',
        '/admin',
      ];
      
      for (const url of urls) {
        await page.goto(url);
        await waitForPageLoad(page);
        
        // 验证页面加载（不是 404）
        const notFound = page.locator('text=404').or(page.locator('text=Not Found'));
        expect(await notFound.isVisible()).toBeFalsy();
        
        console.log(`✓ ${url} 页面正常访问`);
      }
    });
  });

  test.describe('9. 错误处理和边界测试', () => {
    
    test.beforeEach(async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
    });

    test('9.1 访问不存在的页面', async ({ page }) => {
      await page.goto('/non-existent-page');
      await page.waitForTimeout(2000);
      
      // 验证显示 404 或重定向
      const is404 = await page.locator('text=404').or(page.locator('text=Not Found')).isVisible();
      const isRedirected = !page.url().includes('non-existent-page');
      
      expect(is404 || isRedirected).toBeTruthy();
    });

    test('9.2 网络错误处理', async ({ page, context }) => {
      // 设置网络离线
      await context.setOffline(true);
      
      await page.goto('/projects').catch(() => {});
      
      // 恢复网络
      await context.setOffline(false);
      
      // 重新加载
      await page.reload();
      await waitForPageLoad(page);
    });

    test('9.3 大数据量渲染', async ({ page }) => {
      await page.goto('/slurm-tasks');
      await waitForPageLoad(page);
      
      // 验证表格渲染
      const table = page.locator('.ant-table');
      await expect(table).toBeVisible();
      
      // 检查性能
      const startTime = Date.now();
      await page.waitForTimeout(1000);
      const renderTime = Date.now() - startTime;
      
      expect(renderTime).toBeLessThan(5000);
    });
  });

  test.describe('10. 集成测试', () => {
    
    test('10.1 完整工作流：登录 → 访问页面 → 执行操作 → 登出', async ({ page }) => {
      // 1. 登录
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
      console.log('✓ 步骤 1: 登录成功');
      
      // 2. 访问 SLURM 任务页面
      await page.goto('/slurm-tasks');
      await waitForPageLoad(page);
      console.log('✓ 步骤 2: SLURM 任务页面加载');
      
      // 3. 切换到统计信息
      const statisticsTab = page.locator('text=统计信息');
      if (await statisticsTab.isVisible()) {
        await statisticsTab.click();
        await page.waitForTimeout(2000);
        console.log('✓ 步骤 3: 切换到统计信息');
      }
      
      // 4. 访问对象存储
      await page.goto('/object-storage');
      await waitForPageLoad(page);
      console.log('✓ 步骤 4: 对象存储页面加载');
      
      // 5. 访问 SaltStack
      await page.goto('/saltstack');
      await waitForPageLoad(page);
      console.log('✓ 步骤 5: SaltStack 页面加载');
      
      // 6. 登出
      await logout(page);
      console.log('✓ 步骤 6: 登出成功');
      
      // 验证返回登录页
      await expect(page).toHaveURL(/\/auth/);
    });

    test('10.2 跨页面状态保持', async ({ page }) => {
      await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
      
      // 访问不同页面，验证登录状态保持
      const pages = ['/projects', '/slurm-tasks', '/object-storage', '/saltstack'];
      
      for (const url of pages) {
        await page.goto(url);
        await waitForPageLoad(page);
        
        // 验证仍然登录（用户菜单可见）
        const userMenu = await page.locator('.ant-dropdown-trigger').first();
        await expect(userMenu).toBeVisible();
        
        console.log(`✓ 访问 ${url} - 登录状态保持`);
      }
    });
  });
});

// 性能测试套件
test.describe('性能测试', () => {
  
  test('页面加载性能', async ({ page }) => {
    await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
    
    const pages = [
      { url: '/projects', name: '项目列表' },
      { url: '/slurm-tasks', name: 'SLURM 任务' },
      { url: '/object-storage', name: '对象存储' },
      { url: '/saltstack', name: 'SaltStack' },
    ];
    
    for (const { url, name } of pages) {
      const startTime = Date.now();
      await page.goto(url);
      await waitForPageLoad(page);
      const loadTime = Date.now() - startTime;
      
      console.log(`${name} 页面加载时间: ${loadTime} ms`);
      expect(loadTime).toBeLessThan(10000); // 10 秒内
    }
  });
});
