const { test, expect } = require('@playwright/test');

test.describe('Monitoring Page - Check for 404 Message', () => {
  test('Check if page contains "404" or "页面不存在" message', async ({ page }) => {
    // Login first
    await page.goto('http://192.168.0.200:8080/login', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    await page.waitForSelector('input[type="text"], input[name="username"]', { timeout: 10000 });
    await page.locator('input[type="text"], input[name="username"]').first().fill('admin');
    await page.locator('input[type="password"], input[name="password"]').first().fill('admin123');
    await page.locator('button[type="submit"]').click();
    
    await page.waitForURL(/\/(projects|monitoring)/, { timeout: 10000 });

    // Navigate to monitoring page
    await page.goto('http://192.168.0.200:8080/monitoring', { 
      waitUntil: 'load',
      timeout: 30000 
    });

    // Wait for page to load
    await page.waitForTimeout(3000);

    // Get main page content
    const mainPageContent = await page.content();
    console.log('\n=== 主页面检查 ===');
    
    // Check for 404 in main page
    if (mainPageContent.includes('404') || mainPageContent.includes('页面不存在')) {
      console.log('❌ 主页面包含 404 或 页面不存在');
      console.log('相关内容:', mainPageContent.substring(
        mainPageContent.indexOf('404') - 50,
        mainPageContent.indexOf('404') + 100
      ));
    } else {
      console.log('✅ 主页面没有发现 404 或 页面不存在');
    }

    // Check all frames
    const frames = page.frames();
    console.log(`\n=== iframe 检查 (共 ${frames.length} 个) ===`);

    for (let i = 0; i < frames.length; i++) {
      const frame = frames[i];
      const frameUrl = frame.url();
      console.log(`\nFrame ${i + 1}: ${frameUrl}`);

      try {
        // Get frame content
        const frameContent = await frame.content();
        
        // Check for 404 message
        if (frameContent.includes('404') || frameContent.includes('页面不存在') || frameContent.includes('Not Found')) {
          console.log(`❌ 此 frame 包含 404 相关信息`);
          
          // Try to find the exact text
          const bodyText = await frame.locator('body').innerText().catch(() => '');
          if (bodyText.includes('404') || bodyText.includes('页面不存在')) {
            console.log(`可见文本内容:`);
            console.log(bodyText.substring(0, 500));
          }
        } else {
          console.log(`✅ 此 frame 未发现 404 信息`);
        }
      } catch (e) {
        console.log(`⚠️  无法读取此 frame 的内容: ${e.message}`);
      }
    }

    // Take screenshot for debugging
    await page.screenshot({ 
      path: 'test-screenshots/monitoring-404-message-check.png',
      fullPage: true 
    });
    console.log('\n📸 截图已保存: test-screenshots/monitoring-404-message-check.png');

    // Check if Nightingale iframe exists and is accessible
    const nightingaleIframe = frames.find(f => 
      f.url().includes('nightingale') || f.url().includes('n9e')
    );

    if (nightingaleIframe) {
      console.log('\n=== Nightingale iframe 详细检查 ===');
      console.log(`URL: ${nightingaleIframe.url()}`);
      
      try {
        // Wait a bit for content to load
        await page.waitForTimeout(2000);
        
        // Try to get the title or some content
        const title = await nightingaleIframe.title().catch(() => '');
        console.log(`标题: ${title}`);
        
        // Get visible text
        const bodyText = await nightingaleIframe.locator('body').innerText().catch(() => '');
        if (bodyText) {
          console.log(`\n可见内容预览 (前 500 字符):`);
          console.log(bodyText.substring(0, 500));
          
          // Check for 404 in visible text
          if (bodyText.includes('404') || bodyText.includes('页面不存在')) {
            console.log('\n❌❌❌ 发现问题：Nightingale iframe 中包含 "404" 或 "页面不存在" 信息');
            
            // Find and print the context around the 404 message
            const index404 = bodyText.indexOf('404') >= 0 ? bodyText.indexOf('404') : bodyText.indexOf('页面不存在');
            if (index404 >= 0) {
              console.log('\n完整错误信息上下文:');
              console.log(bodyText.substring(Math.max(0, index404 - 100), Math.min(bodyText.length, index404 + 200)));
            }
          }
        }
      } catch (e) {
        console.log(`获取内容时出错: ${e.message}`);
      }
    } else {
      console.log('\n❌ 未找到 Nightingale iframe');
    }
  });
});
