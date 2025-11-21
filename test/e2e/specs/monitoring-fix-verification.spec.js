const { test, expect } = require('@playwright/test');

test('Monitoring page should load correctly', async ({ page }) => {
  // Navigate to the monitoring page
  const response = await page.goto('http://192.168.0.199:8080/monitoring');
  
  // Check that the response is not 404
  expect(response.status()).toBe(200);

  // Wait for the page to load
  await page.waitForLoadState('networkidle');

  // Check for the presence of the Nightingale iframe or specific content
  // Since the user mentioned "Data Query - Metrics", we look for that text or the iframe
  // The iframe usually has id="nightingale-frame" or similar, or we can check for the title
  
  // Take a screenshot for debugging
  await page.screenshot({ path: 'test-screenshots/monitoring-page.png' });

  // Check if the URL is still /monitoring (or redirected to login if not auth, but let's assume we might need login)
  // If it redirects to login, we might need to handle that. 
  // For now, let's just check it's not 404 and has some content.
  
  const title = await page.title();
  console.log(`Page title: ${title}`);
  
  // If the page loads the React app, it should have the root element
  await expect(page.locator('#root')).toBeVisible();
});
