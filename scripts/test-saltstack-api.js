const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext();
  const page = await context.newPage();

  // 登录
  console.log('Navigating to login page...');
  await page.goto('http://192.168.3.101:8080/login', { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(2000);
  
  console.log('Filling login form...');
  await page.fill('input[name="username"]', 'admin', { timeout: 60000 });
  await page.fill('input[name="password"]', 'admin123', { timeout: 60000 });
  await page.click('button[type="submit"]', { timeout: 60000 });
  
  console.log('Waiting for redirect...');
  await page.waitForURL('**/projects', { timeout: 30000 }).catch(() => {
    console.log('Still on login page or redirect failed, continuing anyway...');
  });
  await page.waitForTimeout(3000);

  // 导航到 SaltStack 页面
  console.log('Navigating to SaltStack page...');
  await page.goto('http://192.168.3.101:8080/saltstack', { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(3000);

  // 在控制台中获取 node-metrics 数据
  console.log('Fetching node-metrics...');
  const metricsData = await page.evaluate(async () => {
    try {
      const response = await fetch('/api/saltstack/node-metrics');
      const data = await response.json();
      console.log('node-metrics response:', JSON.stringify(data, null, 2));
      return data;
    } catch (error) {
      console.error('Error fetching node-metrics:', error);
      return { error: error.message };
    }
  });

  console.log('\n=== Node Metrics Data ===');
  console.log(JSON.stringify(metricsData, null, 2));

  // 获取 minions 数据
  console.log('\nFetching minions...');
  const minionsData = await page.evaluate(async () => {
    try {
      const response = await fetch('/api/saltstack/minions');
      const data = await response.json();
      console.log('minions response:', JSON.stringify(data, null, 2));
      return data;
    } catch (error) {
      console.error('Error fetching minions:', error);
      return { error: error.message };
    }
  });

  console.log('\n=== Minions Data ===');
  console.log(JSON.stringify(minionsData, null, 2));

  console.log('\n=== Waiting 30 seconds before exit ===');
  await page.waitForTimeout(30000);
  
  await browser.close();
})();
