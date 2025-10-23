const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

test.describe('SLURM Page - Remove Duplicate Task Button', () => {
  test('verify task management button is removed from source', async () => {
    const filePath = path.join(__dirname, '../../../src/frontend/src/pages/SlurmScalingPage.js');
    const content = fs.readFileSync(filePath, 'utf-8');

    console.log('\n=== Verifying SLURM Page Top Buttons ===\n');

    // 检查是否移除了"任务管理"按钮
    const hasTaskButton = content.includes('任务管理') && content.includes('/slurm-tasks');
    console.log('1. Has "任务管理" button:', hasTaskButton ? '❌ YES (BAD)' : '✅ NO (GOOD)');
    expect(hasTaskButton).toBe(false);

    // 检查是否还有其他必要的按钮
    const hasRefreshButton = content.includes('刷新');
    console.log('2. Has "刷新" button:', hasRefreshButton ? '✅ YES' : '❌ NO');
    expect(hasRefreshButton).toBe(true);

    const hasScaleUpButton = content.includes('扩容节点');
    console.log('3. Has "扩容节点" button:', hasScaleUpButton ? '✅ YES' : '❌ NO');
    expect(hasScaleUpButton).toBe(true);

    const hasSaltStackButton = content.includes('SaltStack 命令');
    console.log('4. Has "SaltStack 命令" button:', hasSaltStackButton ? '✅ YES' : '❌ NO');
    expect(hasSaltStackButton).toBe(true);

    console.log('\n✅ All button verifications passed!\n');
  });

  test('verify page title is simple', async () => {
    const filePath = path.join(__dirname, '../../../src/frontend/src/pages/SlurmScalingPage.js');
    const content = fs.readFileSync(filePath, 'utf-8');

    console.log('\n=== Verifying SLURM Page Title ===\n');

    // 检查标题
    const hasSimpleTitle = content.includes('<ClusterOutlined /> SLURM');
    console.log('Page title is simple "SLURM":', hasSimpleTitle ? '✅ YES' : '❌ NO');
    expect(hasSimpleTitle).toBe(true);

    const hasComplexTitle = content.includes('SLURM 弹性扩缩容管理');
    console.log('Has complex title:', hasComplexTitle ? '❌ YES (BAD)' : '✅ NO (GOOD)');
    expect(hasComplexTitle).toBe(false);

    console.log('\n✅ Title verification passed!\n');
  });
});

test.describe('SLURM Page - Summary', () => {
  test('print fix summary', async () => {
    console.log('\n');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('  SLURM 页面修复总结');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('');
    console.log('✅ 已完成的修复:');
    console.log('  1. 移除页面顶部的"任务管理"按钮');
    console.log('     - 避免与主标题"SLURM"产生混淆');
    console.log('     - 减少页面顶部的视觉冗余');
    console.log('');
    console.log('  2. 保留的按钮:');
    console.log('     - 刷新按钮');
    console.log('     - 扩容节点按钮');
    console.log('     - SaltStack 命令按钮');
    console.log('');
    console.log('  3. 页面标题:');
    console.log('     - 简洁的 "SLURM" 标题（带集群图标）');
    console.log('');
    console.log('📦 构建前端容器:');
    console.log('  docker-compose build frontend');
    console.log('  docker-compose up -d frontend');
    console.log('');
    console.log('🔍 验证:');
    console.log('  访问 http://192.168.0.200:8080/slurm');
    console.log('  预期: 页面顶部只显示 "SLURM" 标题和必要的操作按钮');
    console.log('  预期: 没有"任务管理"或"Slurm任务"相关的冗余按钮');
    console.log('');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('\n');
  });
});
