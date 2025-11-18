const { test, expect } = require('@playwright/test');

test.describe('SLURM Page Header Check', () => {
  test.beforeEach(async ({ page }) => {
    // Login
    await page.goto('http://192.168.0.200:8080/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('http://192.168.0.200:8080/', { timeout: 5000 });
  });

  test('should check SLURM page header content', async ({ page }) => {
    // Navigate to SLURM page
    await page.goto('http://192.168.0.200:8080/slurm');
    await page.waitForLoadState('networkidle');

    // Take a screenshot of the page
    await page.screenshot({ 
      path: 'test-screenshots/slurm-page-before-fix.png',
      fullPage: true 
    });

    // Get page HTML to analyze
    const bodyHTML = await page.locator('body').innerHTML();
    console.log('Page contains iframe:', bodyHTML.includes('<iframe'));

    // Check for duplicate text
    const pageText = await page.locator('body').textContent();
    console.log('\n=== Page Text Content ===');
    console.log(pageText);

    // Check iframe source
    const iframes = await page.locator('iframe').all();
    console.log(`\n=== Found ${iframes.length} iframe(s) ===`);
    
    for (let i = 0; i < iframes.length; i++) {
      const src = await iframes[i].getAttribute('src');
      const title = await iframes[i].getAttribute('title');
      console.log(`Iframe ${i + 1}:`);
      console.log(`  src: ${src}`);
      console.log(`  title: ${title}`);
    }

    // Check for breadcrumb or header
    const headers = await page.locator('h1, h2, h3, .ant-page-header-heading-title, .breadcrumb').all();
    console.log(`\n=== Found ${headers.length} header(s) ===`);
    
    for (let i = 0; i < headers.length; i++) {
      const text = await headers[i].textContent();
      console.log(`Header ${i + 1}: "${text}"`);
    }

    // Check for any element containing "slurm任务"
    const slurmTaskElements = page.locator('text=slurm任务');
    const count = await slurmTaskElements.count();
    console.log(`\n=== Found ${count} element(s) containing "slurm任务" ===`);
    
    if (count > 0) {
      for (let i = 0; i < count; i++) {
        const element = slurmTaskElements.nth(i);
        const text = await element.textContent();
        const tagName = await element.evaluate(el => el.tagName);
        console.log(`Element ${i + 1}: <${tagName}> "${text}"`);
      }
    }
  });

  test('should analyze SLURM page structure', async ({ page }) => {
    await page.goto('http://192.168.0.200:8080/slurm');
    await page.waitForLoadState('networkidle');

    // Get main content container
    const mainContent = page.locator('.ant-layout-content, main, [role="main"]');
    const html = await mainContent.innerHTML();
    
    console.log('\n=== Main Content HTML (first 2000 chars) ===');
    console.log(html.substring(0, 2000));

    // Check route component
    const evaluate = await page.evaluate(() => {
      const pathname = window.location.pathname;
      const title = document.title;
      return { pathname, title };
    });
    
    console.log('\n=== Page Info ===');
    console.log('Pathname:', evaluate.pathname);
    console.log('Title:', evaluate.title);
  });
});
