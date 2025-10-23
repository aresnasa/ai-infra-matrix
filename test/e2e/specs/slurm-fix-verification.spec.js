const { test, expect } = require('@playwright/test');

test.describe('SLURM Page Fix - Quick Verification', () => {
  test('verify SLURM page source code changes', async () => {
    // 这个测试验证源代码已经修复
    const fs = require('fs');
    const path = require('path');
    
    const filePath = path.join(__dirname, '../../../src/frontend/src/pages/SlurmScalingPage.js');
    const content = fs.readFileSync(filePath, 'utf-8');
    
    console.log('\n=== Source Code Verification ===');
    
    // 检查是否已移除 SlurmTaskBar import
    const hasSlurmTaskBarImport = content.includes("import SlurmTaskBar from '../components/SlurmTaskBar'");
    console.log('Has SlurmTaskBar import:', hasSlurmTaskBarImport ? '❌ YES (BAD)' : '✅ NO (GOOD)');
    expect(hasSlurmTaskBarImport).toBe(false);
    
    // 检查是否已移除 SlurmTaskBar 使用
    const usesSlurmTaskBar = content.includes('<SlurmTaskBar');
    console.log('Uses SlurmTaskBar component:', usesSlurmTaskBar ? '❌ YES (BAD)' : '✅ NO (GOOD)');
    expect(usesSlurmTaskBar).toBe(false);
    
    // 检查标题是否更改为简单的 "SLURM"
    const hasSimpleTitle = content.includes('<ClusterOutlined /> SLURM');
    const hasComplexTitle = content.includes('SLURM 弹性扩缩容管理');
    
    console.log('Has simple "SLURM" title:', hasSimpleTitle ? '✅ YES (GOOD)' : '❌ NO (BAD)');
    console.log('Has complex title text:', hasComplexTitle ? '❌ YES (BAD)' : '✅ NO (GOOD)');
    
    expect(hasSimpleTitle).toBe(true);
    expect(hasComplexTitle).toBe(false);
    
    console.log('\n✅ All source code fixes verified!');
  });

  test('check monitoring iframe configuration', async () => {
    const fs = require('fs');
    const path = require('path');
    
    const filePath = path.join(__dirname, '../../../src/frontend/src/pages/SlurmScalingPage.js');
    const content = fs.readFileSync(filePath, 'utf-8');
    
    console.log('\n=== Monitoring Dashboard Configuration ===');
    
    // 检查是否包含监控仪表板 tab
    const hasMonitoringTab = content.includes('监控仪表板') && content.includes('BarChartOutlined');
    console.log('Has monitoring dashboard tab:', hasMonitoringTab ? '✅ YES' : '❌ NO');
    expect(hasMonitoringTab).toBe(true);
    
    // 检查 iframe 配置
    const hasNightingaleIframe = content.includes('id="slurm-dashboard-iframe"') && 
                                 content.includes('/nightingale/');
    console.log('Has Nightingale iframe:', hasNightingaleIframe ? '✅ YES' : '❌ NO');
    expect(hasNightingaleIframe).toBe(true);
    
    console.log('\n✅ Monitoring configuration verified!');
  });
});

test.describe('SLURM Page Summary', () => {
  test('print fix summary', async () => {
    console.log('\n');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('  SLURM 页面修复摘要');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('');
    console.log('修复内容:');
    console.log('  1. ✅ 移除不存在的 SlurmTaskBar 组件引用');
    console.log('  2. ✅ 简化页面标题: "SLURM 弹性扩缩容管理" → "SLURM"');
    console.log('  3. ✅ 保留所有功能性标签页（节点管理、作业队列等）');
    console.log('  4. ✅ 保留 Nightingale 监控仪表板 iframe');
    console.log('');
    console.log('修改文件:');
    console.log('  - src/frontend/src/pages/SlurmScalingPage.js');
    console.log('');
    console.log('下一步操作:');
    console.log('  1. 重新构建前端容器:');
    console.log('     docker-compose build frontend');
    console.log('');
    console.log('  2. 重启前端服务:');
    console.log('     docker-compose up -d frontend');
    console.log('');
    console.log('  3. 访问页面验证:');
    console.log('     http://192.168.0.200:8080/slurm');
    console.log('');
    console.log('预期结果:');
    console.log('  - 页面顶部标题显示为 "SLURM"（不是 "SLURM 弹性扩缩容管理"）');
    console.log('  - 无控制台错误（SlurmTaskBar 组件已移除）');
    console.log('  - 所有标签页正常显示');
    console.log('  - 监控仪表板标签页加载 Nightingale iframe');
    console.log('');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('');
  });
});
