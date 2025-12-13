/**
 * 监控页面 iframe 语言切换测试
 * 测试 Nightingale iframe 能否正确响应语言切换
 */
const { test, expect } = require('@playwright/test');

const BASE_URL = process.env.TEST_BASE_URL || 'http://192.168.3.101:8080';

test.describe('监控页面 iframe 语言切换测试', () => {
  test.setTimeout(120000);
  
  test.beforeEach(async ({ page }) => {
    // 先访问项目页面确保已登录
    await page.goto(`${BASE_URL}/project`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await page.waitForTimeout(2000);
    
    // 如果跳转到登录页，执行登录
    if (page.url().includes('/login')) {
      console.log('[Test] Logging in...');
      await page.fill('input[id="username"]', 'admin');
      await page.fill('input[id="password"]', 'admin123');
      await page.click('button[type="submit"]');
      await page.waitForURL('**/project**', { timeout: 15000 });
      console.log('[Test] Login successful');
    }
    
    // 确保登录完成
    await page.waitForTimeout(2000);
  });

  test('访问监控页面并检查 iframe URL 参数', async ({ page }) => {
    // 导航到监控页面
    console.log('[Test] Navigating to monitoring page...');
    await page.goto(`${BASE_URL}/monitoring`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    
    // 等待更长时间让 React 渲染完成
    await page.waitForTimeout(8000);
    
    // 截图当前状态
    await page.screenshot({ path: 'test-screenshots/monitoring-language-01-initial.png', fullPage: true });
    
    // 打印页面内容以调试
    const pageTitle = await page.title();
    console.log('[Test] Page title:', pageTitle);
    
    const pageUrl = page.url();
    console.log('[Test] Current URL:', pageUrl);
    
    // 检查页面是否有错误提示
    const errorAlert = page.locator('.ant-alert-error');
    if (await errorAlert.isVisible()) {
      const errorText = await errorAlert.textContent();
      console.log('[Test] Error alert:', errorText);
    }
    
    // 检查加载状态
    const spinner = page.locator('.ant-spin');
    const spinnerVisible = await spinner.isVisible();
    console.log('[Test] Spinner visible:', spinnerVisible);
    
    // 等待 iframe 出现
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    const iframeCount = await iframe.count();
    console.log('[Test] iframe count:', iframeCount);
    
    if (iframeCount > 0) {
      await expect(iframe).toBeVisible({ timeout: 30000 });
      
      // 获取 iframe 的 src 属性
      const iframeSrc = await iframe.getAttribute('src');
      console.log('[Test] iframe src:', iframeSrc);
      
      // 检查 URL 是否包含语言参数
      expect(iframeSrc).toContain('lang=');
      expect(iframeSrc).toContain('themeMode=');
      
      // 解析参数
      const url = new URL(iframeSrc);
      const langParam = url.searchParams.get('lang');
      const themeParam = url.searchParams.get('themeMode');
      console.log('[Test] lang:', langParam, 'themeMode:', themeParam);
    } else {
      console.log('[Test] No iframe found on the page');
      
      // 检查页面 HTML 内容
      const pageContent = await page.content();
      console.log('[Test] Page contains "iframe":', pageContent.includes('iframe'));
      console.log('[Test] Page contains "Nightingale":', pageContent.includes('Nightingale'));
      console.log('[Test] Page contains "monitoring":', pageContent.includes('monitoring'));
      
      // 检查 React 渲染的卡片
      const card = page.locator('.ant-card');
      const cardCount = await card.count();
      console.log('[Test] Card count:', cardCount);
      
      if (cardCount > 0) {
        const cardTitle = await card.first().locator('.ant-card-head-title').textContent();
        console.log('[Test] Card title:', cardTitle);
      }
    }
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/monitoring-language-02-iframe-loaded.png', fullPage: true });
  });

  test('检测 iframe 内容和背景样式', async ({ page }) => {
    // 导航到监控页面
    console.log('[Test] Navigating to monitoring page...');
    await page.goto(`${BASE_URL}/monitoring`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await page.waitForTimeout(10000);
    
    // 获取 iframe
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    const iframeCount = await iframe.count();
    console.log('[Test] iframe count:', iframeCount);
    
    if (iframeCount > 0) {
      const iframeSrc = await iframe.getAttribute('src');
      console.log('[Test] iframe src:', iframeSrc);
      
      // 获取 iframe 样式
      const iframeStyle = await iframe.getAttribute('style');
      console.log('[Test] iframe style:', iframeStyle);
      
      // 检查 Card 组件样式
      const card = page.locator('.ant-card').first();
      if (await card.isVisible()) {
        const cardBg = await card.evaluate(el => window.getComputedStyle(el).backgroundColor);
        console.log('[Test] Card background:', cardBg);
      }
    } else {
      // 打印更多调试信息
      const allIframes = page.locator('iframe');
      const allIframeCount = await allIframes.count();
      console.log('[Test] Total iframes on page:', allIframeCount);
      
      for (let i = 0; i < allIframeCount; i++) {
        const iframeSrc = await allIframes.nth(i).getAttribute('src');
        const iframeTitle = await allIframes.nth(i).getAttribute('title');
        console.log(`[Test] iframe ${i}: title="${iframeTitle}", src="${iframeSrc}"`);
      }
    }
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/monitoring-language-03-styles.png', fullPage: true });
  });

  test('中英文来回切换测试 - 第一次中文 → 英文 → 再次中文', async ({ page }) => {
    // 辅助函数：切换语言
    const switchLanguage = async (targetLang) => {
      console.log(`[Test] Switching to ${targetLang}...`);
      
      // 找到语言切换按钮（通常是国际化图标）
      const langBtn = page.locator('[data-testid="language-switch"], .ant-dropdown-trigger:has(.anticon-global), button:has(.anticon-global), .language-switch');
      
      // 尝试多种选择器
      let clicked = false;
      const selectors = [
        '[data-testid="language-switch"]',
        '.ant-dropdown-trigger:has(.anticon-global)',
        'button:has(.anticon-global)',
        '.language-switch',
        '.anticon-global',
        '[class*="language"]',
        // Header 中的下拉菜单
        'header .ant-dropdown-trigger',
        '.ant-layout-header .ant-dropdown-trigger'
      ];
      
      for (const selector of selectors) {
        const btn = page.locator(selector);
        if (await btn.count() > 0 && await btn.first().isVisible()) {
          await btn.first().click();
          clicked = true;
          console.log(`[Test] Clicked language button with selector: ${selector}`);
          break;
        }
      }
      
      if (!clicked) {
        console.log('[Test] Could not find language switch button, trying direct navigation');
        // 直接通过 URL 参数切换
        const currentUrl = page.url();
        const newUrl = targetLang === 'zh' 
          ? currentUrl.replace(/lang=en/g, 'lang=zh').replace(/([?&])$/, '')
          : currentUrl.replace(/lang=zh/g, 'lang=en').replace(/([?&])$/, '');
        
        if (!newUrl.includes('lang=')) {
          const separator = newUrl.includes('?') ? '&' : '?';
          await page.goto(`${newUrl}${separator}lang=${targetLang}`, { waitUntil: 'domcontentloaded' });
        } else {
          await page.goto(newUrl, { waitUntil: 'domcontentloaded' });
        }
      } else {
        // 等待下拉菜单出现
        await page.waitForTimeout(500);
        
        // 点击目标语言选项
        const langOption = targetLang === 'zh' 
          ? page.locator('text=中文, text=简体中文, text=Chinese').first()
          : page.locator('text=English, text=英文, text=EN').first();
        
        if (await langOption.isVisible()) {
          await langOption.click();
        } else {
          // 通过 localStorage 设置语言并刷新
          await page.evaluate((lang) => {
            localStorage.setItem('i18n_lang', lang);
            localStorage.setItem('language', lang === 'zh' ? 'zh_CN' : 'en_US');
          }, targetLang);
          await page.reload();
        }
      }
      
      await page.waitForTimeout(3000);
    };

    // 辅助函数：等待 iframe 加载完成
    const waitForIframeReady = async (timeout = 15000) => {
      const startTime = Date.now();
      const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
      
      while (Date.now() - startTime < timeout) {
        // 检查 iframe 是否存在且可见
        const iframeCount = await iframe.count();
        if (iframeCount > 0) {
          const isVisible = await iframe.first().isVisible({ timeout: 1000 }).catch(() => false);
          if (isVisible) {
            console.log('[Test] iframe is now visible');
            return true;
          }
        }
        
        // 检查加载状态（Spin 组件消失）
        const spinner = page.locator('.ant-spin');
        const isSpinnerVisible = await spinner.isVisible({ timeout: 500 }).catch(() => false);
        if (!isSpinnerVisible && iframeCount > 0) {
          console.log('[Test] Loading spinner disappeared and iframe exists');
          return true;
        }
        
        await page.waitForTimeout(500);
      }
      
      console.log('[Test] iframe did not become ready within timeout');
      return false;
    };

    // 辅助函数：检查 iframe 中 Nightingale 的语言
    const checkIframeLang = async (expectedLang) => {
      const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
      
      if (await iframe.count() === 0) {
        console.log('[Test] No iframe found');
        return null;
      }
      
      const iframeSrc = await iframe.getAttribute('src');
      console.log(`[Test] iframe src: ${iframeSrc}`);
      
      // 获取 iframe 的 frame 对象
      const frame = page.frameLocator('iframe[title="Nightingale Monitoring"]');
      
      // 检查 iframe 内的 localStorage
      const iframeElement = await iframe.elementHandle();
      const contentFrame = await iframeElement.contentFrame();
      
      if (contentFrame) {
        const n9eLang = await contentFrame.evaluate(() => {
          return localStorage.getItem('language');
        });
        console.log(`[Test] Nightingale localStorage language: ${n9eLang}`);
        
        // 验证语言是否正确
        const expectedN9eLang = expectedLang === 'zh' ? 'zh_CN' : 'en_US';
        return n9eLang === expectedN9eLang;
      }
      
      return null;
    };

    // 步骤1：中文访问监控页面
    console.log('[Test] Step 1: Visit monitoring page in Chinese');
    await page.evaluate(() => {
      localStorage.setItem('i18n_lang', 'zh');
      localStorage.setItem('language', 'zh_CN');
    });
    await page.goto(`${BASE_URL}/monitoring?lang=zh`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    
    // 等待 iframe 加载
    const ready1 = await waitForIframeReady();
    console.log(`[Test] First iframe ready: ${ready1}`);
    
    await page.screenshot({ path: 'test-screenshots/monitoring-lang-switch-01-chinese-first.png', fullPage: true });
    
    // 检查中文状态
    let zhResult1 = await checkIframeLang('zh');
    console.log(`[Test] First Chinese visit - language correct: ${zhResult1}`);
    
    // 步骤2：切换到英文
    console.log('[Test] Step 2: Switch to English');
    await page.evaluate(() => {
      localStorage.setItem('i18n_lang', 'en');
      localStorage.setItem('language', 'en_US');
    });
    await page.goto(`${BASE_URL}/monitoring?lang=en`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    
    // 等待 iframe 加载
    const ready2 = await waitForIframeReady();
    console.log(`[Test] Second iframe ready: ${ready2}`);
    
    await page.screenshot({ path: 'test-screenshots/monitoring-lang-switch-02-english.png', fullPage: true });
    
    // 检查英文状态
    let enResult = await checkIframeLang('en');
    console.log(`[Test] English visit - language correct: ${enResult}`);
    
    // 步骤3：再切回中文（这是问题发生的场景）
    console.log('[Test] Step 3: Switch back to Chinese (this is where the bug occurs)');
    await page.evaluate(() => {
      localStorage.setItem('i18n_lang', 'zh');
      localStorage.setItem('language', 'zh_CN');
      // 清除可能的 reload 标记
      sessionStorage.removeItem('n9e_lang_reload');
    });
    await page.goto(`${BASE_URL}/monitoring?lang=zh`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    
    // 等待 iframe 加载
    const ready3 = await waitForIframeReady();
    console.log(`[Test] Third iframe ready: ${ready3}`);
    
    await page.screenshot({ path: 'test-screenshots/monitoring-lang-switch-03-chinese-again.png', fullPage: true });
    
    // 检查第二次中文状态
    let zhResult2 = await checkIframeLang('zh');
    console.log(`[Test] Second Chinese visit - language correct: ${zhResult2}`);
    
    // 打印 iframe src 用于调试
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    if (await iframe.count() > 0) {
      const finalSrc = await iframe.getAttribute('src');
      console.log(`[Test] Final iframe src: ${finalSrc}`);
      
      // 验证 URL 参数
      expect(finalSrc).toContain('lang=zh');
    }
    
    // 记录测试结果
    console.log(`[Test] Test Results Summary:`);
    console.log(`[Test] Step 1 (First Chinese): ${zhResult1}`);
    console.log(`[Test] Step 2 (English): ${enResult}`);
    console.log(`[Test] Step 3 (Second Chinese): ${zhResult2}`);
  });

  test('监控页面语言同步脚本执行验证', async ({ page }) => {
    // 这个测试验证 nginx 注入的语言同步脚本是否正确执行
    
    // 设置语言为中文
    await page.evaluate(() => {
      localStorage.setItem('i18n_lang', 'zh');
      localStorage.setItem('language', 'zh_CN');
    });
    
    // 访问监控页面
    await page.goto(`${BASE_URL}/monitoring?lang=zh`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await page.waitForTimeout(5000);
    
    // 获取 iframe
    const iframe = page.locator('iframe[title="Nightingale Monitoring"]');
    
    if (await iframe.count() > 0) {
      const iframeElement = await iframe.elementHandle();
      const contentFrame = await iframeElement.contentFrame();
      
      if (contentFrame) {
        // 检查语言同步结果
        const langState = await contentFrame.evaluate(() => {
          return {
            localStorage_language: localStorage.getItem('language'),
            localStorage_theme: localStorage.getItem('theme'),
            sessionStorage_reload: sessionStorage.getItem('n9e_lang_reload')
          };
        });
        
        console.log('[Test] iframe language state:', JSON.stringify(langState, null, 2));
        
        // 验证语言已同步
        expect(langState.localStorage_language).toBe('zh_CN');
      }
    }
    
    await page.screenshot({ path: 'test-screenshots/monitoring-lang-switch-04-sync-verify.png', fullPage: true });
  });
});
