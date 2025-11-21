const { test, expect } = require('@playwright/test');

test('Analyze Monitoring Page 404', async ({ page }) => {
  // 0. Login as admin
  console.log('Logging in as admin...');
  await page.goto('http://192.168.0.199:8080/login');
  await page.fill('input[name="username"]', 'admin');
  await page.fill('input[name="password"]', 'admin123'); // Assuming default password
  await page.click('button[type="submit"]');
  await page.waitForURL('**/projects'); // Wait for redirect to projects
  console.log('Login successful');

  // 1. Check the main monitoring page
  console.log('Navigating to /monitoring...');
  const response = await page.goto('http://192.168.0.199:8080/monitoring');
  console.log(`Main page status: ${response.status()}`);
  
  // Take a screenshot of the main page
  await page.waitForTimeout(5000); // Wait for iframe to load
  await page.screenshot({ path: 'test-screenshots/monitoring-page-view.png' });

  // 2. Check if the iframe exists
  const iframeElement = page.locator('iframe');
  const iframeCount = await iframeElement.count();
  console.log(`Iframe count: ${iframeCount}`);

  if (iframeCount > 0) {
    const src = await iframeElement.getAttribute('src');
    console.log(`Iframe src: ${src}`);

    // 3. Check the iframe content directly
    console.log(`Navigating to iframe src: ${src}...`);
    const iframeResponse = await page.goto(src);
    console.log(`Iframe response status: ${iframeResponse.status()}`);
    
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'test-screenshots/nightingale-iframe-view.png' });
    
    // Check for specific 404 text in the iframe content
    const content = await page.content();
    if (content.includes('404') || content.includes('Not Found')) {
      console.log('Found "404" or "Not Found" in iframe content');
    } else {
      console.log('Did not find explicit 404 text in iframe content');
    }
  } else {
    console.log('No iframe found on the page');
  }
});
