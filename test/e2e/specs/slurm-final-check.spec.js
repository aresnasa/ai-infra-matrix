const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

test.describe('SLURM Page - Complete Fix Verification', () => {
  test('comprehensive source code check', async () => {
    const filePath = path.join(__dirname, '../../../src/frontend/src/pages/SlurmScalingPage.js');
    const content = fs.readFileSync(filePath, 'utf-8');

    console.log('\n');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('  SLURM 页面完整修复验证');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('');

    // 1. 检查移除的内容
    console.log('❌ 已移除的内容:');
    
    const checks = [
      {
        name: 'SlurmTaskBar 组件引用',
        test: !content.includes("import SlurmTaskBar from '../components/SlurmTaskBar'"),
        expected: true
      },
      {
        name: 'SlurmTaskBar 组件使用',
        test: !content.includes('<SlurmTaskBar'),
        expected: true
      },
      {
        name: '复杂标题"SLURM 弹性扩缩容管理"',
        test: !content.includes('SLURM 弹性扩缩容管理'),
        expected: true
      },
      {
        name: '"任务管理"按钮（避免与SLURM混淆）',
        test: !(content.includes('任务管理') && content.includes('/slurm-tasks')),
        expected: true
      }
    ];

    checks.forEach((check, index) => {
      const status = check.test === check.expected ? '✅' : '❌';
      console.log(`  ${status} ${index + 1}. ${check.name}`);
      expect(check.test).toBe(check.expected);
    });

    console.log('');
    console.log('✅ 保留的内容:');

    const keeps = [
      {
        name: '简洁的"SLURM"标题',
        test: content.includes('<ClusterOutlined /> SLURM'),
        expected: true
      },
      {
        name: '刷新按钮',
        test: content.includes('刷新'),
        expected: true
      },
      {
        name: '扩容节点按钮',
        test: content.includes('扩容节点'),
        expected: true
      },
      {
        name: 'SaltStack 命令按钮',
        test: content.includes('SaltStack 命令'),
        expected: true
      },
      {
        name: '节点管理标签页',
        test: content.includes('节点管理'),
        expected: true
      },
      {
        name: '作业队列标签页',
        test: content.includes('作业队列'),
        expected: true
      },
      {
        name: '监控仪表板标签页',
        test: content.includes('监控仪表板') && content.includes('nightingale'),
        expected: true
      }
    ];

    keeps.forEach((keep, index) => {
      const status = keep.test === keep.expected ? '✅' : '❌';
      console.log(`  ${status} ${index + 1}. ${keep.name}`);
      expect(keep.test).toBe(keep.expected);
    });

    console.log('');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('  修复效果总结');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('');
    console.log('📝 问题描述:');
    console.log('  "Slurm 和 Slurm任务两个页顶部标签还是存在"');
    console.log('');
    console.log('🔧 解决方案:');
    console.log('  1. 移除了不存在的 SlurmTaskBar 组件引用');
    console.log('  2. 简化页面标题为 "SLURM"');
    console.log('  3. 移除了容易混淆的"任务管理"按钮');
    console.log('  4. 保留所有功能性标签页和必要按钮');
    console.log('');
    console.log('✨ 预期结果:');
    console.log('  - 页面顶部显示清晰的 "SLURM" 标题');
    console.log('  - 只有必要的操作按钮（刷新、扩容节点、SaltStack命令）');
    console.log('  - 无"任务管理"或重复的"Slurm任务"按钮');
    console.log('  - 页面内标签页完整（节点管理、作业队列、监控等）');
    console.log('');
    console.log('📦 下一步操作:');
    console.log('  docker-compose build frontend');
    console.log('  docker-compose up -d frontend');
    console.log('');
    console.log('🔍 访问验证:');
    console.log('  http://192.168.0.200:8080/slurm');
    console.log('');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('\n');
  });
});
