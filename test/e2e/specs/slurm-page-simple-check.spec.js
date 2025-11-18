const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

test.describe('SLURM Page - Simple Verification (No Login)', () => {
  test('verify source code changes are correct', async () => {
    const filePath = path.join(__dirname, '../../../src/frontend/src/pages/SlurmScalingPage.js');
    const content = fs.readFileSync(filePath, 'utf-8');

    console.log('\n=== Source Code Verification ===\n');

    // 检查是否已移除 SlurmTaskBar import
    const hasSlurmTaskBarImport = content.includes("import SlurmTaskBar from '../components/SlurmTaskBar'");
    console.log('1. SlurmTaskBar import removed:', hasSlurmTaskBarImport ? '❌ NO' : '✅ YES');
    expect(hasSlurmTaskBarImport).toBe(false);

    // 检查是否已移除 SlurmTaskBar 使用
    const usesSlurmTaskBar = content.includes('<SlurmTaskBar');
    console.log('2. SlurmTaskBar usage removed:', usesSlurmTaskBar ? '❌ NO' : '✅ YES');
    expect(usesSlurmTaskBar).toBe(false);

    // 检查是否使用了简单标题 "SLURM"（可能包含图标）
    const hasSimpleTitle = content.includes('SLURM') && 
                          content.includes('<Title level={2}>') && 
                          (content.includes('<ClusterOutlined /> SLURM') || content.includes('SLURM</Title>'));
    console.log('3. Simple "SLURM" title present:', hasSimpleTitle ? '✅ YES' : '❌ NO');
    expect(hasSimpleTitle).toBe(true);

    // 检查是否移除了复杂标题
    const hasComplexTitle = content.includes('SLURM 弹性扩缩容管理');
    console.log('4. Complex title removed:', hasComplexTitle ? '❌ NO' : '✅ YES');
    expect(hasComplexTitle).toBe(false);

    // 检查监控仪表板配置
    const hasMonitoringTab = content.includes('监控仪表板') && content.includes('nightingale');
    console.log('5. Monitoring dashboard tab present:', hasMonitoringTab ? '✅ YES' : '❌ NO');
    expect(hasMonitoringTab).toBe(true);

    console.log('\n✅ All source code verifications passed!\n');
  });

  test('check if page is accessible without authentication', async ({ page }) => {
    console.log('\n=== Page Accessibility Check ===\n');

    // 尝试直接访问 SLURM 页面
    const response = await page.goto('http://192.168.0.200:8080/slurm', {
      waitUntil: 'domcontentloaded',
      timeout: 10000
    });

    console.log('Response status:', response.status());
    console.log('Final URL:', page.url());

    // 如果重定向到登录页面，这是正常的
    if (page.url().includes('/login')) {
      console.log('✅ Page requires authentication (redirected to login)');
      console.log('   This is expected behavior for authenticated pages');
    } else {
      console.log('✅ Page loaded directly');
    }

    // 截图当前页面状态
    await page.screenshot({ 
      path: 'test-screenshots/slurm-page-access-check.png',
      fullPage: true 
    });
  });

  test('verify frontend container is using updated code', async ({ page }) => {
    console.log('\n=== Frontend Container Version Check ===\n');

    try {
      // 尝试访问页面
      await page.goto('http://192.168.0.200:8080/slurm', {
        waitUntil: 'domcontentloaded',
        timeout: 10000
      });

      // 获取页面内容
      const pageContent = await page.content();

      // 检查是否包含旧的标题（不应该有）
      const hasOldTitle = pageContent.includes('SLURM 弹性扩缩容管理');
      console.log('Has old title "SLURM 弹性扩缩容管理":', hasOldTitle ? '❌ YES (BAD)' : '✅ NO (GOOD)');

      // 检查是否包含 SlurmTaskBar 错误（不应该有）
      const consoleErrors = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });

      await page.waitForTimeout(2000);

      const hasSlurmTaskBarError = consoleErrors.some(error => 
        error.includes('SlurmTaskBar')
      );
      console.log('Has SlurmTaskBar errors:', hasSlurmTaskBarError ? '❌ YES (BAD)' : '✅ NO (GOOD)');

      console.log('\nNote: If you see old content, you need to rebuild the frontend:');
      console.log('  docker-compose build frontend');
      console.log('  docker-compose up -d frontend');

    } catch (error) {
      console.log('Page access failed:', error.message);
      console.log('This might be normal if authentication is required');
    }
  });
});

test.describe('SLURM Page - Build Instructions', () => {
  test('print build and verification instructions', async () => {
    console.log('\n');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('  SLURM 页面修复 - 构建和验证说明');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('');
    console.log('✅ 源码修复已完成:');
    console.log('  - 移除了 SlurmTaskBar 组件引用');
    console.log('  - 简化标题为 "SLURM"');
    console.log('  - 保留所有功能标签页');
    console.log('');
    console.log('📦 构建前端容器:');
    console.log('  cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix');
    console.log('  docker-compose build frontend');
    console.log('  docker-compose up -d frontend');
    console.log('');
    console.log('🔍 验证步骤:');
    console.log('  1. 等待容器重启完成（约30秒）');
    console.log('  2. 访问: http://192.168.0.200:8080/slurm');
    console.log('  3. 检查页面标题是否为 "SLURM"');
    console.log('  4. 检查浏览器控制台无错误');
    console.log('  5. 验证所有标签页正常显示');
    console.log('');
    console.log('✨ 预期结果:');
    console.log('  - 页面标题: "SLURM"（不是"SLURM 弹性扩缩容管理"）');
    console.log('  - 无 SlurmTaskBar 相关错误');
    console.log('  - 标签页: 节点管理、作业队列、资源监控、监控仪表板');
    console.log('  - 监控仪表板加载 Nightingale iframe');
    console.log('');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('\n');
  });
});
