import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://192.168.0.200:8080';

test.describe('监控页面重定向测试', () => {
  
  test('检查 /monitoring 的重定向和最终页面', async ({ page }) => {
    console.log('\n=== 测试 /monitoring 重定向 ===');
    
    // 访问 /monitoring
    const response = await page.goto(`${BASE_URL}/monitoring`, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    console.log(`响应状态: ${response?.status()}`);
    console.log(`最终 URL: ${page.url()}`);
    
    // 等待页面加载
    await page.waitForTimeout(2000);
    
    // 截图
    await page.screenshot({ 
      path: 'test-results/monitoring-redirect.png', 
      fullPage: true 
    });
    
    // 获取页面内容
    const bodyText = await page.locator('body').textContent();
    console.log(`页面文本长度: ${bodyText?.length || 0}`);
    
    // 检查是否有 404 错误
    const hasNotFound = bodyText?.includes('你访问的页面不存在') || bodyText?.includes('404');
    console.log(`是否显示404错误: ${hasNotFound}`);
    
    if (hasNotFound) {
      console.log('\n❌ 页面显示404错误');
      console.log('页面内容预览:');
      console.log(bodyText?.substring(0, 500));
      
      // 检查 ant-result 元素
      const resultElement = page.locator('.ant-result');
      const resultCount = await resultElement.count();
      console.log(`找到 ${resultCount} 个 .ant-result 元素`);
      
      if (resultCount > 0) {
        const subtitle = await page.locator('.ant-result-subtitle').textContent();
        console.log(`错误提示: ${subtitle}`);
      }
    } else {
      console.log('\n✓ 页面正常加载，无404错误');
    }
    
    // 检查是否成功重定向到 /metric/explorer
    const currentUrl = page.url();
    if (currentUrl.includes('/metric/explorer')) {
      console.log('✓ 成功重定向到 /metric/explorer');
    } else {
      console.log(`⚠️ 未重定向到 /metric/explorer，当前URL: ${currentUrl}`);
    }
    
    // 检查页面内容
    console.log('\n=== 页面元素检查 ===');
    
    // 检查是否有 Nightingale 相关内容
    const hasNightingale = bodyText?.toLowerCase().includes('nightingale') || 
                          bodyText?.includes('夜莺') ||
                          bodyText?.includes('监控');
    console.log(`包含监控相关内容: ${hasNightingale}`);
    
    // 检查 iframe
    const iframeCount = await page.locator('iframe').count();
    console.log(`iframe 数量: ${iframeCount}`);
    
    // 断言：不应该显示404错误
    expect(bodyText).not.toContain('你访问的页面不存在');
    expect(bodyText).not.toContain('ant-result-subtitle');
  });
});
