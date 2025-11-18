const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

test.describe('Navigation - Remove Slurm任务 Menu Item', () => {
  test('verify backend navigation config removed slurm-tasks', async () => {
    const filePath = path.join(__dirname, '../../../src/backend/internal/controllers/navigation.go');
    const content = fs.readFileSync(filePath, 'utf-8');

    console.log('\n=== Backend Navigation Configuration Check ===\n');

    // 检查是否移除了 slurm-tasks 配置
    const hasSlurmTasksConfig = content.includes('slurm-tasks') && content.includes('Slurm任务');
    console.log('1. Has "slurm-tasks" navigation config:', hasSlurmTasksConfig ? '❌ YES (BAD)' : '✅ NO (GOOD)');
    expect(hasSlurmTasksConfig).toBe(false);

    // 检查 SLURM 配置是否存在
    const hasSlurmConfig = content.includes('"slurm"') && content.includes('"/slurm"');
    console.log('2. Has "slurm" navigation config:', hasSlurmConfig ? '✅ YES' : '❌ NO');
    expect(hasSlurmConfig).toBe(true);

    // 检查 SaltStack 配置是否存在
    const hasSaltStackConfig = content.includes('saltstack') && content.includes('SaltStack');
    console.log('3. Has "saltstack" navigation config:', hasSaltStackConfig ? '✅ YES' : '❌ NO');
    expect(hasSaltStackConfig).toBe(true);

    // 检查 Order 是否正确调整
    const saltStackOrderCheck = content.includes('Order:   7') && content.includes('"saltstack"');
    console.log('4. SaltStack Order updated to 7:', saltStackOrderCheck ? '✅ YES' : '❌ NO');
    expect(saltStackOrderCheck).toBe(true);

    console.log('\n✅ Backend navigation configuration verified!\n');
  });

  test('verify frontend CustomizableNavigation does not have slurm-tasks', async () => {
    const filePath = path.join(__dirname, '../../../src/frontend/src/components/CustomizableNavigation.js');
    const content = fs.readFileSync(filePath, 'utf-8');

    console.log('\n=== Frontend Navigation Configuration Check ===\n');

    // 检查前端是否有 slurm-tasks 配置
    const hasSlurmTasksInFrontend = content.includes("id: 'slurm-tasks'") || content.includes("key: '/slurm-tasks'");
    console.log('1. Frontend has "slurm-tasks" config:', hasSlurmTasksInFrontend ? '❌ YES (BAD)' : '✅ NO (GOOD)');
    expect(hasSlurmTasksInFrontend).toBe(false);

    // 检查前端是否有 SLURM 配置
    const hasSlurmInFrontend = content.includes("id: 'slurm'") && content.includes("key: '/slurm'");
    console.log('2. Frontend has "slurm" config:', hasSlurmInFrontend ? '✅ YES' : '❌ NO');
    expect(hasSlurmInFrontend).toBe(true);

    console.log('\n✅ Frontend navigation configuration verified!\n');
  });
});

test.describe('Navigation Fix Summary', () => {
  test('print complete fix summary', async () => {
    console.log('\n');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('  导航菜单修复总结 - 移除"Slurm任务"');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('');
    console.log('📝 问题描述:');
    console.log('  顶部导航菜单中同时存在"SLURM"和"Slurm任务"两个菜单项');
    console.log('  HTML: <span class="ant-menu-title-content">Slurm任务</span>');
    console.log('');
    console.log('🔧 修复方案:');
    console.log('  1. 后端: 从 navigation.go 中移除 slurm-tasks 配置项');
    console.log('  2. 后端: 调整 SaltStack 的 Order 从 8 到 7');
    console.log('  3. 前端: SlurmScalingPage.js 已移除"任务管理"按钮');
    console.log('');
    console.log('✅ 已完成的修复:');
    console.log('  ✅ 移除后端 DefaultNavigationItems 中的 slurm-tasks 项');
    console.log('  ✅ 保留 SLURM 主菜单项');
    console.log('  ✅ 移除页面内部的"任务管理"按钮');
    console.log('  ✅ 调整其他菜单项的顺序');
    console.log('');
    console.log('📦 构建步骤:');
    console.log('  1. 重新构建后端:');
    console.log('     docker-compose build backend');
    console.log('     docker-compose up -d backend');
    console.log('');
    console.log('  2. 清除用户导航配置缓存（可选）:');
    console.log('     - 用户可以在导航栏右侧点击设置按钮');
    console.log('     - 点击"重置默认"按钮');
    console.log('     - 或者手动删除数据库中的用户导航配置');
    console.log('');
    console.log('🔍 验证步骤:');
    console.log('  1. 重启后端容器');
    console.log('  2. 访问 http://192.168.0.200:8080');
    console.log('  3. 登录后查看顶部导航菜单');
    console.log('  4. 确认只有"SLURM"菜单项，没有"Slurm任务"');
    console.log('');
    console.log('✨ 预期结果:');
    console.log('  - 顶部导航栏只显示"SLURM"一个菜单项');
    console.log('  - 点击"SLURM"进入 /slurm 页面');
    console.log('  - 页面顶部只显示"SLURM"标题');
    console.log('  - 没有"任务管理"或"Slurm任务"相关按钮/菜单');
    console.log('');
    console.log('📌 注意事项:');
    console.log('  - /slurm-tasks 路由保留在 App.js 中（用于特定场景访问）');
    console.log('  - 只是从顶部导航菜单中移除，不影响功能');
    console.log('  - 用户仍可通过页面内的链接访问任务详情');
    console.log('');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('\n');
  });
});
