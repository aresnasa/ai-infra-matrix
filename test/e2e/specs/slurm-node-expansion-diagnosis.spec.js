/**
 * SLURM 节点扩容和服务诊断测试
 * 
 * 测试目标：
 * 1. 测试通过 Web 界面扩容节点
 * 2. 检查节点状态（idle* 表示 slurmd 未连接）
 * 3. 诊断 slurmd 服务启动问题
 * 4. 验证配置文件和目录权限
 */

const { test, expect } = require('@playwright/test');
const { exec } = require('child_process');
const { promisify } = require('util');

const execPromise = promisify(exec);

const BASE_URL = process.env.BASE_URL || 'http://192.168.3.98:8080';

// 辅助函数：执行 SSH 命令
async function execSSHCommand(host, command) {
  const sshCmd = `sshpass -p aiinfra2024 ssh -o StrictHostKeyChecking=no root@${host} "${command}"`;
  try {
    const { stdout, stderr } = await execPromise(sshCmd);
    return { stdout: stdout.trim(), stderr: stderr.trim(), success: true };
  } catch (error) {
    return { 
      stdout: error.stdout ? error.stdout.trim() : '', 
      stderr: error.stderr ? error.stderr.trim() : '', 
      success: false,
      error: error.message 
    };
  }
}

// 辅助函数：检查节点 slurmd 状态
async function checkSlurmdStatus(nodeName) {
  console.log(`\n检查节点 ${nodeName} 的 slurmd 状态...`);
  
  const checks = {
    systemctl: await execSSHCommand(nodeName, 'systemctl status slurmd --no-pager'),
    running: await execSSHCommand(nodeName, 'pgrep -x slurmd'),
    logs: await execSSHCommand(nodeName, 'journalctl -u slurmd -n 30 --no-pager'),
    config: await execSSHCommand(nodeName, 'ls -la /etc/slurm/slurm.conf'),
    dirs: await execSSHCommand(nodeName, 'ls -la /var/run/slurm /var/log/slurm /var/spool/slurmd 2>&1'),
    munge: await execSSHCommand(nodeName, 'systemctl is-active munge'),
  };
  
  return checks;
}

// 辅助函数：获取 SLURM 节点状态
async function getSlurmNodeStatus() {
  try {
    const { stdout } = await execPromise('docker exec ai-infra-slurm-master sinfo -Nel');
    return stdout;
  } catch (error) {
    console.error('获取节点状态失败:', error.message);
    return null;
  }
}

test.describe('SLURM 节点扩容诊断测试', () => {
  
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(`${BASE_URL}/`, { timeout: 10000 });
  });

  test('应该能够通过 Web 界面扩容节点', async ({ page }) => {
    console.log('\n=== 测试节点扩容功能 ===');
    
    // 导航到 SLURM 页面
    await page.goto(`${BASE_URL}/slurm`);
    await page.waitForLoadState('networkidle');
    
    // 等待页面加载
    await page.waitForSelector('h1:has-text("SLURM")', { timeout: 10000 });
    console.log('✓ SLURM 页面已加载');
    
    // 查找扩容按钮
    const expandButton = page.locator('button:has-text("扩容")').first();
    await expect(expandButton).toBeVisible({ timeout: 5000 });
    console.log('✓ 找到扩容按钮');
    
    // 截图：扩容前
    await page.screenshot({ 
      path: 'test-screenshots/slurm-before-expansion.png',
      fullPage: true 
    });
    
    // 点击扩容按钮
    await expandButton.click();
    await page.waitForTimeout(1000);
    
    // 等待扩容对话框出现
    const dialog = page.locator('.ant-modal, .el-dialog, [role="dialog"]').first();
    await expect(dialog).toBeVisible({ timeout: 5000 });
    console.log('✓ 扩容对话框已打开');
    
    // 截图：扩容对话框
    await page.screenshot({ 
      path: 'test-screenshots/slurm-expansion-dialog.png',
      fullPage: true 
    });
    
    // 查找节点输入框或选择器
    // 尝试多种可能的选择器
    const nodeInput = await page.locator('input[placeholder*="节点"], input[placeholder*="主机"], textarea[placeholder*="节点"]').first();
    
    if (await nodeInput.isVisible()) {
      // 输入测试节点
      const testNodes = 'test-ssh01,test-ssh02,test-ssh03';
      await nodeInput.fill(testNodes);
      console.log(`✓ 输入测试节点: ${testNodes}`);
      
      await page.waitForTimeout(500);
      
      // 查找并点击确认/提交按钮
      const submitButton = page.locator('button:has-text("确定"), button:has-text("提交"), button:has-text("添加"), button[type="submit"]').first();
      await submitButton.click();
      console.log('✓ 点击提交按钮');
      
      // 等待操作完成
      await page.waitForTimeout(3000);
      
      // 检查是否有成功消息
      const successMessage = page.locator('.ant-message-success, .el-message--success, :has-text("成功")');
      if (await successMessage.isVisible({ timeout: 2000 }).catch(() => false)) {
        console.log('✓ 看到成功消息');
      }
      
      // 截图：扩容后
      await page.screenshot({ 
        path: 'test-screenshots/slurm-after-expansion.png',
        fullPage: true 
      });
    } else {
      console.log('⚠️  未找到节点输入框，可能需要其他操作步骤');
      
      // 打印对话框内容以便调试
      const dialogContent = await dialog.textContent();
      console.log('对话框内容:', dialogContent);
    }
  });

  test('应该检查扩容后的节点状态', async ({ page }) => {
    console.log('\n=== 检查节点状态 ===');
    
    // 获取 SLURM 节点状态
    const nodeStatus = await getSlurmNodeStatus();
    console.log('\n当前节点状态:');
    console.log(nodeStatus);
    
    // 检查是否有 idle* 状态的节点
    if (nodeStatus && nodeStatus.includes('idle*')) {
      console.log('\n⚠️  发现 idle* 状态的节点（slurmd 未连接）');
      
      // 提取节点名称
      const idleNodes = nodeStatus.split('\n')
        .filter(line => line.includes('idle*'))
        .map(line => line.split(/\s+/)[0])
        .filter(name => name && name !== 'NODELIST');
      
      console.log('idle* 状态节点列表:', idleNodes);
      
      // 逐个诊断节点
      for (const nodeName of idleNodes.slice(0, 3)) { // 限制最多检查3个节点
        console.log(`\n=== 诊断节点: ${nodeName} ===`);
        
        const checks = await checkSlurmdStatus(nodeName);
        
        // 1. systemctl 状态
        console.log('\n1. systemctl 状态:');
        console.log(checks.systemctl.stdout || checks.systemctl.stderr);
        
        // 2. 进程检查
        console.log('\n2. slurmd 进程:');
        if (checks.running.success && checks.running.stdout) {
          console.log(`✓ slurmd 进程运行中 (PID: ${checks.running.stdout})`);
        } else {
          console.log('✗ slurmd 进程未运行');
        }
        
        // 3. 最近日志
        console.log('\n3. 最近日志:');
        console.log(checks.logs.stdout || checks.logs.stderr);
        
        // 4. 配置文件
        console.log('\n4. 配置文件:');
        console.log(checks.config.stdout || checks.config.stderr);
        
        // 5. 目录权限
        console.log('\n5. 目录权限:');
        console.log(checks.dirs.stdout || checks.dirs.stderr);
        
        // 6. munge 状态
        console.log('\n6. munge 状态:');
        console.log(checks.munge.stdout || 'inactive');
        
        // 诊断总结
        console.log('\n--- 诊断总结 ---');
        const issues = [];
        
        if (!checks.running.success || !checks.running.stdout) {
          issues.push('slurmd 进程未运行');
        }
        
        if (!checks.config.success) {
          issues.push('配置文件不存在或无法访问');
        }
        
        if (checks.dirs.stderr && checks.dirs.stderr.includes('No such file or directory')) {
          issues.push('必需目录缺失');
        }
        
        if (checks.munge.stdout !== 'active') {
          issues.push('munge 服务未运行');
        }
        
        if (issues.length > 0) {
          console.log('发现的问题:');
          issues.forEach((issue, i) => console.log(`  ${i + 1}. ${issue}`));
        } else {
          console.log('✓ 未发现明显问题，但 slurmd 仍未连接到 slurmctld');
        }
      }
    } else if (nodeStatus && !nodeStatus.includes('idle*')) {
      console.log('✓ 所有节点状态正常（无 idle* 状态）');
    } else {
      console.log('⚠️  无法获取节点状态');
    }
  });

  test('应该尝试修复 slurmd 服务问题', async ({ page }) => {
    console.log('\n=== 尝试修复 slurmd 服务 ===');
    
    const nodeStatus = await getSlurmNodeStatus();
    
    if (!nodeStatus || !nodeStatus.includes('idle*')) {
      console.log('✓ 没有需要修复的节点');
      return;
    }
    
    // 提取 idle* 节点
    const idleNodes = nodeStatus.split('\n')
      .filter(line => line.includes('idle*'))
      .map(line => line.split(/\s+/)[0])
      .filter(name => name && name !== 'NODELIST');
    
    console.log('需要修复的节点:', idleNodes);
    
    // 修复第一个节点作为示例
    const nodeName = idleNodes[0];
    if (!nodeName) return;
    
    console.log(`\n修复节点: ${nodeName}`);
    
    // 修复步骤 1: 创建必需目录
    console.log('\n步骤 1: 创建必需目录...');
    const createDirs = await execSSHCommand(nodeName, `
      mkdir -p /var/run/slurm /var/log/slurm /var/spool/slurmd && \
      chown -R slurm:slurm /var/run/slurm /var/log/slurm /var/spool/slurmd && \
      chmod 755 /var/run/slurm /var/log/slurm /var/spool/slurmd
    `);
    console.log(createDirs.success ? '✓ 目录创建成功' : '✗ 目录创建失败');
    if (!createDirs.success) {
      console.log('错误:', createDirs.stderr || createDirs.error);
    }
    
    // 修复步骤 2: 检查配置文件
    console.log('\n步骤 2: 检查配置文件...');
    const checkConfig = await execSSHCommand(nodeName, 'cat /etc/slurm/slurm.conf | grep -E "TaskPlugin|ProctrackType"');
    console.log('配置内容:');
    console.log(checkConfig.stdout || checkConfig.stderr);
    
    // 修复步骤 3: 确保 munge 运行
    console.log('\n步骤 3: 启动 munge 服务...');
    const startMunge = await execSSHCommand(nodeName, 'systemctl start munge && systemctl is-active munge');
    console.log(startMunge.success && startMunge.stdout === 'active' ? '✓ munge 已启动' : '✗ munge 启动失败');
    
    // 修复步骤 4: 重启 slurmd
    console.log('\n步骤 4: 重启 slurmd 服务...');
    const restartSlurmd = await execSSHCommand(nodeName, 'systemctl restart slurmd');
    await new Promise(resolve => setTimeout(resolve, 2000)); // 等待 2 秒
    
    // 修复步骤 5: 检查 slurmd 状态
    console.log('\n步骤 5: 检查 slurmd 状态...');
    const statusCheck = await execSSHCommand(nodeName, 'systemctl is-active slurmd');
    console.log('slurmd 状态:', statusCheck.stdout || 'inactive');
    
    if (statusCheck.stdout === 'active') {
      console.log('✓ slurmd 服务已启动');
    } else {
      // 查看错误日志
      console.log('\n查看错误日志:');
      const errorLogs = await execSSHCommand(nodeName, 'journalctl -u slurmd -n 20 --no-pager');
      console.log(errorLogs.stdout || errorLogs.stderr);
    }
    
    // 修复步骤 6: 验证节点状态
    console.log('\n步骤 6: 验证节点状态...');
    await new Promise(resolve => setTimeout(resolve, 3000)); // 等待 3 秒让节点注册
    
    const newStatus = await getSlurmNodeStatus();
    console.log('\n修复后节点状态:');
    console.log(newStatus);
    
    const nodeLineAfter = newStatus?.split('\n').find(line => line.includes(nodeName));
    if (nodeLineAfter) {
      const isFixed = !nodeLineAfter.includes('idle*');
      console.log(isFixed ? `✓ 节点 ${nodeName} 已修复（状态正常）` : `⚠️  节点 ${nodeName} 仍需进一步调查`);
    }
  });

  test('应该生成诊断报告', async ({ page }) => {
    console.log('\n=== 生成诊断报告 ===');
    
    const report = {
      timestamp: new Date().toISOString(),
      nodeStatus: await getSlurmNodeStatus(),
      issues: [],
      recommendations: []
    };
    
    // 分析节点状态
    if (report.nodeStatus) {
      const idleStarCount = (report.nodeStatus.match(/idle\*/g) || []).length;
      const idleCount = (report.nodeStatus.match(/\sidle\s/g) || []).length;
      const downCount = (report.nodeStatus.match(/down/g) || []).length;
      
      console.log('\n节点状态统计:');
      console.log(`  - idle* (slurmd 未连接): ${idleStarCount}`);
      console.log(`  - idle (正常空闲): ${idleCount}`);
      console.log(`  - down (不可用): ${downCount}`);
      
      if (idleStarCount > 0) {
        report.issues.push(`${idleStarCount} 个节点的 slurmd 服务未连接到 slurmctld`);
        report.recommendations.push('检查节点上的 slurmd 服务状态');
        report.recommendations.push('验证 /var/run/slurm, /var/log/slurm, /var/spool/slurmd 目录存在且权限正确');
        report.recommendations.push('确认 munge 服务正常运行');
        report.recommendations.push('检查 /etc/slurm/slurm.conf 配置正确');
      }
      
      if (downCount > 0) {
        report.issues.push(`${downCount} 个节点处于 down 状态`);
        report.recommendations.push('使用 scontrol update NodeName=<node> State=RESUME 恢复节点');
      }
    }
    
    // 输出报告
    console.log('\n=== 诊断报告 ===');
    console.log(JSON.stringify(report, null, 2));
    
    // 保存报告到文件
    const fs = require('fs');
    const reportPath = 'test-screenshots/slurm-diagnosis-report.json';
    fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
    console.log(`\n✓ 诊断报告已保存到: ${reportPath}`);
  });
});
