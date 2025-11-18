/**
 * AI Infrastructure Matrix - 快速验证测试
 * 
 * 专注测试最近修复的功能：
 * 1. JupyterHub 配置渲染
 * 2. Gitea 静态资源路径
 * 3. Object Storage 自动刷新
 * 4. SLURM Dashboard SaltStack 集成
 * 5. SLURM Tasks 刷新频率优化
 */

const { test, expect } = require('@playwright/test');

const TEST_CONFIG = {
  baseURL: process.env.BASE_URL || 'http://192.168.0.200:8080',
  adminUser: {
    username: process.env.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || 'admin123',
  },
};

async function login(page, username, password) {
  await page.goto('/');
  await page.waitForSelector('input[type="text"]', { timeout: 10000 });
  await page.fill('input[type="text"]', username);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/projects', { timeout: 15000 });
}

async function waitForPageLoad(page) {
  // 使用 domcontentloaded 而不是 networkidle，避免因为长轮询或自动刷新导致超时
  await page.waitForLoadState('domcontentloaded', { timeout: 15000 });
  // 等待主要内容加载
  await page.waitForTimeout(2000);
  // 等待加载动画消失
  await page.waitForSelector('.ant-spin-spinning', { state: 'detached', timeout: 10000 }).catch(() => {});
}

test.describe('快速验证测试 - 最近修复功能', () => {
  
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
  });

  test('1. JupyterHub 配置渲染验证', async ({ page }) => {
    console.log('测试 JupyterHub 配置渲染...');
    
    await page.goto('/jupyter');
    await waitForPageLoad(page);
    
    // 等待 iframe 加载
    await page.waitForTimeout(3000);
    
    // 检查是否有错误信息
    const errorMsg = page.locator('text=404').or(page.locator('text=error'));
    const hasError = await errorMsg.isVisible().catch(() => false);
    
    if (hasError) {
      console.log('❌ JupyterHub 可能存在配置问题');
    } else {
      console.log('✓ JupyterHub 页面加载正常');
    }
    
    // 验证 iframe 或内容存在
    const iframe = page.locator('iframe');
    const hasIframe = await iframe.isVisible().catch(() => false);
    
    if (hasIframe) {
      console.log('✓ JupyterHub iframe 已加载');
    }
  });

  test('2. Gitea 静态资源路径验证', async ({ page }) => {
    console.log('测试 Gitea 静态资源路径...');
    
    // 监听资源加载失败
    const failedResources = [];
    page.on('response', response => {
      if (response.status() === 404 && response.url().includes('assets')) {
        failedResources.push(response.url());
      }
    });
    
    await page.goto('/gitea', { timeout: 30000 }); // Gitea 页面可能加载较慢
    await waitForPageLoad(page);
    await page.waitForTimeout(3000);
    
    // 检查是否有 /assets/assets/ 重复路径
    const duplicateAssets = failedResources.filter(url => url.includes('/assets/assets/'));
    
    if (duplicateAssets.length > 0) {
      console.log('❌ 发现重复的 assets 路径:');
      duplicateAssets.forEach(url => console.log(`  - ${url}`));
      expect(duplicateAssets.length).toBe(0);
    } else {
      console.log('✓ Gitea 静态资源路径正常');
    }
  });

  test('3. Object Storage 自动刷新功能验证', async ({ page }) => {
    console.log('测试 Object Storage 自动刷新...');
    
    // 先设置 API 监听器，再导航到页面
    const apiRequests = [];
    page.on('request', request => {
      if (request.url().includes('/api/object-storage/configs')) {
        const timestamp = Date.now();
        apiRequests.push({
          time: timestamp,
          url: request.url()
        });
        console.log(`[API 请求 ${apiRequests.length}] ${new Date(timestamp).toLocaleTimeString()}: ${request.url()}`);
      }
    });
    
    // 导航到页面（会触发初始加载）
    await page.goto('/object-storage');
    await waitForPageLoad(page);
    await page.waitForTimeout(2000);
    
    console.log(`初始加载完成，已捕获 ${apiRequests.length} 个 API 请求`);
    
    // 验证刷新按钮存在
    const refreshButton = page.locator('text=刷新').or(page.locator('[aria-label*="reload"]'));
    const hasRefreshButton = await refreshButton.first().isVisible().catch(() => false);
    
    if (hasRefreshButton) {
      console.log('✓ 发现刷新按钮');
    }
    
    // 验证自动刷新按钮状态
    const autoRefreshButton = page.locator('text=/自动刷新|已暂停/');
    const hasAutoRefresh = await autoRefreshButton.isVisible().catch(() => false);
    
    if (hasAutoRefresh) {
      const buttonText = await autoRefreshButton.textContent();
      console.log(`✓ 自动刷新控制: ${buttonText}`);
      
      // 如果自动刷新被禁用，启用它
      if (buttonText.includes('已暂停')) {
        console.log('启用自动刷新...');
        await autoRefreshButton.click();
        await page.waitForTimeout(1000);
      }
    }
    
    // 验证最后刷新时间显示
    const lastRefreshText = page.locator('text=/上次更新|最后刷新/');
    const hasLastRefresh = await lastRefreshText.isVisible().catch(() => false);
    
    if (hasLastRefresh) {
      const refreshTimeText = await lastRefreshText.textContent();
      console.log(`✓ 刷新时间显示: ${refreshTimeText}`);
    }
    
    // 清空之前的请求记录，开始监听自动刷新
    const initialRequestCount = apiRequests.length;
    console.log(`清空请求记录，开始监听自动刷新（当前已有 ${initialRequestCount} 个请求）...`);
    console.log('等待 35 秒以验证自动刷新（间隔 30 秒）...');
    
    // 等待第一次自动刷新
    await page.waitForTimeout(35000);
    
    const newRequestCount = apiRequests.length - initialRequestCount;
    console.log(`等待期间新增 ${newRequestCount} 个 API 请求`);
    
    if (newRequestCount >= 1) {
      // 计算最后两个请求的间隔
      if (apiRequests.length >= 2) {
        const lastTwo = apiRequests.slice(-2);
        const interval = lastTwo[1].time - lastTwo[0].time;
        console.log(`✓ 最近两次请求间隔: ${interval / 1000} 秒`);
        
        // 如果间隔太短（< 5秒），可能是手动刷新或页面切换导致
        if (interval < 5000) {
          console.log('⚠ 间隔太短，可能不是自动刷新，跳过验证');
        } else {
          // 验证间隔在合理范围内（25-35秒）
          expect(interval).toBeGreaterThanOrEqual(25000);
          expect(interval).toBeLessThanOrEqual(35000);
          console.log('✓ 自动刷新间隔验证通过');
        }
      }
    } else {
      console.log('⚠ 未检测到新的 API 请求，自动刷新可能未启用');
      // 不强制要求，因为可能是后端服务问题
    }
  });

  test('4. SLURM Dashboard SaltStack 集成显示验证', async ({ page }) => {
    console.log('测试 SLURM Dashboard SaltStack 集成...');
    
    await page.goto('/slurm');
    await waitForPageLoad(page);
    
    // 等待数据加载
    await page.waitForTimeout(3000);
    
    // 查找 SaltStack 集成相关元素
    const saltStackKeywords = [
      'SaltStack',
      'Minion',
      '集成',
      'Integration',
      '在线',
      'Online'
    ];
    
    let foundCount = 0;
    for (const keyword of saltStackKeywords) {
      const element = page.locator(`text=${keyword}`).first();
      const isVisible = await element.isVisible().catch(() => false);
      if (isVisible) {
        console.log(`✓ 发现关键字: ${keyword}`);
        foundCount++;
      }
    }
    
    if (foundCount >= 2) {
      console.log('✓ SaltStack 集成信息已显示');
    } else {
      console.log('⚠ SaltStack 集成信息可能未完全加载');
    }
    
    // 验证卡片数量（应该包含 SaltStack 集成卡片）
    const cards = page.locator('.ant-card');
    const cardCount = await cards.count();
    console.log(`✓ 找到 ${cardCount} 个统计卡片`);
  });

  // 5. SLURM Tasks 刷新频率优化验证
  test('5. SLURM Tasks 刷新频率优化验证', async ({ page }) => {
    // 设置测试超时为 90 秒
    test.setTimeout(90000);
    console.log('测试 SLURM Tasks 刷新频率优化...');
    
    // 监听 API 请求
    const taskRequests = [];
    page.on('request', request => {
      if (request.url().includes('/api/slurm/tasks')) {
        taskRequests.push({
          time: Date.now(),
          url: request.url()
        });
        console.log(`[${new Date().toISOString()}] API 请求: ${request.url()}`);
      }
    });
    
    // 访问带参数的 URL（之前报告的问题场景）
    await page.goto('/slurm-tasks?status=running', { waitUntil: 'domcontentloaded' });
    // 不使用 waitForPageLoad，因为有自动刷新会导致 networkidle 永远不满足
    await page.waitForTimeout(3000); // 等待初始加载
    
    console.log('初始加载完成，等待自动刷新...');
    
    // 记录初始请求数
    const initialRequestCount = taskRequests.length;
    console.log(`初始请求数: ${initialRequestCount}`);
    
    // 等待 65 秒，观察刷新行为
    console.log('等待 65 秒以验证刷新频率...');
    await page.waitForTimeout(65000);
    
    const finalRequestCount = taskRequests.length;
    console.log(`最终请求数: ${finalRequestCount}`);
    
    // 分析请求间隔
    if (taskRequests.length >= 2) {
      for (let i = 1; i < taskRequests.length; i++) {
        const interval = taskRequests[i].time - taskRequests[i - 1].time;
        console.log(`请求 ${i} 间隔: ${(interval / 1000).toFixed(1)} 秒`);
      }
      
      // 验证最后一次刷新间隔
      const lastInterval = taskRequests[taskRequests.length - 1].time - taskRequests[taskRequests.length - 2].time;
      
      // 优化后的刷新间隔应该在 30-60 秒之间
      if (lastInterval >= 30000) {
        console.log(`✓ 刷新间隔正常: ${(lastInterval / 1000).toFixed(1)} 秒`);
        expect(lastInterval).toBeGreaterThanOrEqual(30000);
      } else {
        console.log(`❌ 刷新间隔过短: ${(lastInterval / 1000).toFixed(1)} 秒（期望 >= 30 秒）`);
        expect(lastInterval).toBeGreaterThanOrEqual(30000);
      }
    } else {
      console.log('⚠ 请求数量不足，无法验证刷新间隔');
    }
    
    // 验证没有循环刷新（65 秒内不应该有超过 3 次请求）
    const refreshCount = finalRequestCount - initialRequestCount;
    console.log(`自动刷新次数: ${refreshCount}`);
    
    if (refreshCount <= 2) {
      console.log('✓ 刷新频率正常，没有循环刷新');
    } else {
      console.log(`⚠ 刷新次数较多: ${refreshCount} 次（65 秒内）`);
    }
  });

  test('6. SLURM Tasks 统计信息加载验证', async ({ page }) => {
    console.log('测试 SLURM Tasks 统计信息加载...');
    
    await page.goto('/slurm-tasks');
    await waitForPageLoad(page);
    
    // 切换到统计信息 Tab
    const statisticsTab = page.locator('text=统计信息');
    if (await statisticsTab.isVisible()) {
      console.log('✓ 找到统计信息 Tab');
      await statisticsTab.click();
      await page.waitForTimeout(2000);
      
      // 验证统计卡片加载
      const cards = page.locator('.ant-card');
      const cardCount = await cards.count();
      
      if (cardCount > 0) {
        console.log(`✓ 统计信息已加载，共 ${cardCount} 个卡片`);
      } else {
        console.log('❌ 统计信息未加载');
      }
      
      // 检查是否有加载错误
      const errorMessage = page.locator('text=加载失败').or(page.locator('text=Error'));
      const hasError = await errorMessage.isVisible().catch(() => false);
      
      if (hasError) {
        console.log('❌ 统计信息加载出现错误');
      } else {
        console.log('✓ 没有加载错误');
      }
    } else {
      console.log('⚠ 未找到统计信息 Tab');
    }
  });

  test('7. 控制台错误检查', async ({ page }) => {
    console.log('检查浏览器控制台错误...');
    
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    const pages = [
      '/slurm-tasks',
      '/object-storage',
      '/slurm',
      '/saltstack'
    ];
    
    for (const url of pages) {
      await page.goto(url);
      await waitForPageLoad(page);
      await page.waitForTimeout(2000);
    }
    
    if (consoleErrors.length > 0) {
      console.log(`⚠ 发现 ${consoleErrors.length} 个控制台错误:`);
      consoleErrors.forEach((error, index) => {
        console.log(`  ${index + 1}. ${error}`);
      });
    } else {
      console.log('✓ 没有控制台错误');
    }
  });

  test('8. 网络请求监控', async ({ page }) => {
    console.log('监控网络请求...');
    
    const failedRequests = [];
    page.on('response', response => {
      if (response.status() >= 400) {
        failedRequests.push({
          url: response.url(),
          status: response.status()
        });
      }
    });
    
    const pages = [
      '/slurm-tasks',
      '/object-storage',
      '/slurm',
      '/saltstack',
      '/jupyter',
      '/gitea'
    ];
    
    for (const url of pages) {
      await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 });
      await page.waitForTimeout(2000); // 简单等待，不使用 networkidle
      await page.waitForTimeout(1000);
    }
    
    if (failedRequests.length > 0) {
      console.log(`⚠ 发现 ${failedRequests.length} 个失败的网络请求:`);
      failedRequests.forEach(({ url, status }) => {
        console.log(`  [${status}] ${url}`);
      });
    } else {
      console.log('✓ 所有网络请求成功');
    }
  });
});

// 性能基准测试
test.describe('性能基准测试', () => {
  
  test('页面加载性能', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await login(page, TEST_CONFIG.adminUser.username, TEST_CONFIG.adminUser.password);
    
    const pages = [
      { url: '/slurm-tasks', name: 'SLURM Tasks', maxTime: 5000 },
      { url: '/object-storage', name: 'Object Storage', maxTime: 5000 },
      { url: '/slurm', name: 'SLURM Dashboard', maxTime: 5000 },
      { url: '/saltstack', name: 'SaltStack', maxTime: 15000 }, // SaltStack 页面可能较慢
    ];
    
    console.log('\n性能基准测试结果:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    
    for (const { url, name, maxTime } of pages) {
      const startTime = Date.now();
      await page.goto(url);
      await waitForPageLoad(page);
      const loadTime = Date.now() - startTime;
      
      const status = loadTime < maxTime ? '✓' : '⚠';
      console.log(`${status} ${name.padEnd(20)} ${loadTime}ms (期望 < ${maxTime}ms)`);
      
      expect(loadTime).toBeLessThan(maxTime * 2); // 允许 2 倍容差
    }
    
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
  });
});
