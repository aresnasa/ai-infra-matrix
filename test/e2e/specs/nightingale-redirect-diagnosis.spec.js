const { test, expect } = require('@playwright/test');

test.describe('Nightingale URL Redirect Loop Diagnosis', () => {
  test('check /monitoring for redirect loops', async ({ page }) => {
    console.log('🔍 Diagnosing /monitoring redirect loop...\n');
    
    // Track all navigations
    const navigations = [];
    page.on('framenavigated', frame => {
      if (frame === page.mainFrame()) {
        navigations.push({
          url: frame.url(),
          timestamp: Date.now()
        });
      }
    });
    
    // Track console errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    try {
      console.log('📍 Step 1: Navigating to http://192.168.18.114:8080/monitoring');
      
      // Set a timeout to detect infinite loops
      await page.goto('http://192.168.18.114:8080/monitoring', { 
        waitUntil: 'networkidle',
        timeout: 10000 
      });
      
      await page.waitForTimeout(3000);
      
      console.log(`\n📊 Navigation history (${navigations.length} navigations):`);
      navigations.forEach((nav, i) => {
        console.log(`   ${i + 1}. ${nav.url}`);
      });
      
      // Check for redirect loop
      if (navigations.length > 5) {
        console.log('\n⚠️  WARNING: Too many navigations detected (possible redirect loop)');
        
        // Check if same URL appears multiple times
        const urlCounts = {};
        navigations.forEach(nav => {
          urlCounts[nav.url] = (urlCounts[nav.url] || 0) + 1;
        });
        
        console.log('\n📈 URL frequency:');
        Object.entries(urlCounts).forEach(([url, count]) => {
          console.log(`   ${url}: ${count} times ${count > 2 ? '❌ LOOP!' : ''}`);
        });
      }
      
      // Check current URL
      const finalUrl = page.url();
      console.log(`\n📍 Final URL: ${finalUrl}`);
      
      // Take screenshot
      await page.screenshot({ path: 'test-screenshots/monitoring-redirect-diagnosis.png', fullPage: true });
      console.log('📸 Screenshot saved: monitoring-redirect-diagnosis.png');
      
      // Check if it's the monitoring page with iframe
      const hasIframe = await page.locator('iframe[title*="Nightingale"]').count() > 0;
      console.log(`\n🖼️  Nightingale iframe present: ${hasIframe ? '✓ Yes' : '✗ No'}`);
      
      if (hasIframe) {
        const iframeSrc = await page.locator('iframe[title*="Nightingale"]').getAttribute('src');
        console.log(`   iframe src: ${iframeSrc}`);
      }
      
      // Check console errors
      if (consoleErrors.length > 0) {
        console.log(`\n❌ Console errors (${consoleErrors.length}):`);
        consoleErrors.slice(0, 5).forEach(err => {
          console.log(`   - ${err}`);
        });
      }
      
      // Check if React app loaded
      const hasReactRoot = await page.locator('#root').count() > 0;
      console.log(`\n⚛️  React root element: ${hasReactRoot ? '✓ Yes' : '✗ No'}`);
      
      const rootContent = await page.locator('#root').textContent();
      const hasContent = rootContent && rootContent.trim().length > 0;
      console.log(`   Root has content: ${hasContent ? '✓ Yes' : '✗ No (React may not have loaded)'}`);
      
    } catch (error) {
      console.error('\n❌ Navigation failed:', error.message);
      
      console.log(`\n📊 Navigations before error (${navigations.length}):`);
      navigations.forEach((nav, i) => {
        console.log(`   ${i + 1}. ${nav.url}`);
      });
      
      await page.screenshot({ path: 'test-screenshots/monitoring-error.png', fullPage: true });
      console.log('📸 Error screenshot saved');
    }
  });

  test('check /nightingale/ direct access', async ({ page }) => {
    console.log('\n🔍 Testing /nightingale/ direct access...\n');
    
    try {
      await page.goto('http://192.168.18.114:8080/nightingale/', { 
        waitUntil: 'networkidle',
        timeout: 10000 
      });
      
      await page.waitForTimeout(2000);
      
      const finalUrl = page.url();
      console.log(`📍 Final URL: ${finalUrl}`);
      
      // Check if it's Nightingale login page or dashboard
      const hasLoginForm = await page.locator('input[type="password"]').count() > 0;
      const pageContent = await page.locator('body').textContent();
      const isNightingale = pageContent.includes('Nightingale') || pageContent.includes('夜莺');
      
      console.log(`🔐 Login form present: ${hasLoginForm ? '✓ Yes' : '✗ No'}`);
      console.log(`🦉 Nightingale content: ${isNightingale ? '✓ Yes' : '✗ No'}`);
      
      await page.screenshot({ path: 'test-screenshots/nightingale-direct.png', fullPage: true });
      console.log('📸 Screenshot saved: nightingale-direct.png');
      
      if (hasLoginForm) {
        console.log('\n✅ /nightingale/ is working correctly (shows login page)');
      } else if (isNightingale) {
        console.log('\n✅ /nightingale/ is working correctly (shows Nightingale)');
      } else {
        console.log('\n❌ /nightingale/ is NOT showing Nightingale content');
      }
      
    } catch (error) {
      console.error('\n❌ Failed to load /nightingale/:', error.message);
    }
  });

  test('check nginx sub_filter rewriting', async ({ page }) => {
    console.log('\n🔍 Checking nginx sub_filter rewriting...\n');
    
    const { exec } = require('child_process');
    const { promisify } = require('util');
    const execAsync = promisify(exec);
    
    try {
      // Get the HTML from /nightingale/
      const { stdout } = await execAsync(
        `curl -s http://192.168.18.114:8080/nightingale/ | head -100`
      );
      
      console.log('📄 HTML from /nightingale/ (first 100 lines):');
      
      // Check for path rewriting
      const hasNightingalePaths = stdout.includes('="/nightingale/') || stdout.includes("='/nightingale/");
      const hasRootPaths = stdout.match(/["|']=\/(?!nightingale)/);
      
      console.log(`   Contains "/nightingale/" paths: ${hasNightingalePaths ? '✓ Yes (sub_filter working)' : '✗ No'}`);
      console.log(`   Contains root "/" paths: ${hasRootPaths ? '⚠️  Yes (may cause issues)' : '✓ No (good)'}`);
      
      // Show some examples
      const nightingalePathMatches = stdout.match(/["|']=["']?\/nightingale\/[^"'\s]*/g);
      if (nightingalePathMatches) {
        console.log('\n   Sample rewritten paths:');
        nightingalePathMatches.slice(0, 5).forEach(match => {
          console.log(`   - ${match}`);
        });
      }
      
    } catch (error) {
      console.error('❌ Error checking HTML:', error.message);
    }
  });
});
