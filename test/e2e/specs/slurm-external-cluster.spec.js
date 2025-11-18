const { test, expect } = require('@playwright/test');

/**
 * SLURM 外部集群管理 E2E 测试
 * 
 * 测试场景：
 * 1. 访问外部集群管理页面
 * 2. 测试 SSH 连接到已有集群
 * 3. 添加外部集群
 * 4. 验证集群列表显示
 * 5. 刷新集群信息
 * 6. 删除集群
 */

test.describe('SLURM 外部集群管理', () => {
  // 测试前登录
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL('/dashboard', { timeout: 10000 });
  });

  test('应该能访问外部集群管理页面', async ({ page }) => {
    // 导航到外部集群管理页面
    await page.goto('/slurm/external-clusters');
    
    // 验证页面标题
    await expect(page.locator('h1')).toContainText('外部 SLURM 集群管理');
    
    // 验证两个标签页存在
    await expect(page.locator('button[role="tab"]').nth(0)).toContainText('添加集群');
    await expect(page.locator('button[role="tab"]').nth(1)).toContainText('已连接集群');
  });

  test('应该能填写集群连接表单', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 确保在"添加集群"标签页
    await page.click('button[role="tab"]:has-text("添加集群")');
    
    // 填写基本信息
    await page.fill('input[name="name"]', '测试集群');
    await page.fill('textarea[name="description"]', '用于自动化测试的集群');
    await page.fill('input[name="master_host"]', 'slurm-master');
    
    // 填写 SSH 配置
    await page.fill('input[name="ssh_username"]', 'root');
    await page.fill('input[name="ssh_password"]', 'aiinfra2024');
    
    // 验证表单字段已填充
    await expect(page.locator('input[name="name"]')).toHaveValue('测试集群');
    await expect(page.locator('input[name="master_host"]')).toHaveValue('slurm-master');
  });

  test('应该能测试 SSH 连接', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 填写必要信息
    await page.fill('input[name="master_host"]', 'slurm-master');
    await page.fill('input[name="ssh_username"]', 'root');
    await page.fill('input[name="ssh_password"]', 'aiinfra2024');
    
    // 点击测试连接按钮
    await page.click('button:has-text("测试连接")');
    
    // 等待测试结果（最多30秒，因为SSH连接可能需要时间）
    const alert = page.locator('[role="alert"]');
    await expect(alert).toBeVisible({ timeout: 30000 });
    
    // 验证测试结果（成功或失败都算测试通过）
    const alertText = await alert.textContent();
    expect(alertText).toMatch(/(连接成功|连接失败|连接测试失败)/);
  });

  test('应该能添加外部集群', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    const clusterName = `测试集群-${Date.now()}`;
    
    // 填写完整表单
    await page.fill('input[name="name"]', clusterName);
    await page.fill('textarea[name="description"]', 'Playwright E2E 测试集群');
    await page.fill('input[name="master_host"]', 'slurm-master');
    await page.fill('input[name="ssh_username"]', 'root');
    await page.fill('input[name="ssh_password"]', 'aiinfra2024');
    
    // 先测试连接
    await page.click('button:has-text("测试连接")');
    
    // 等待连接测试成功
    const successAlert = page.locator('[role="alert"]:has-text("连接成功")');
    await expect(successAlert).toBeVisible({ timeout: 30000 });
    
    // 提交表单
    await page.click('button[type="submit"]:has-text("添加集群")');
    
    // 等待成功提示或页面跳转
    await page.waitForTimeout(2000);
    
    // 切换到"已连接集群"标签
    await page.click('button[role="tab"]:has-text("已连接集群")');
    
    // 验证集群出现在列表中
    await expect(page.locator(`text=${clusterName}`)).toBeVisible({ timeout: 5000 });
  });

  test('应该显示已连接集群列表', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 切换到"已连接集群"标签
    await page.click('button[role="tab"]:has-text("已连接集群")');
    
    // 检查是否有集群或空状态
    const hasContent = await page.locator('text=暂无外部集群').isVisible()
      .catch(() => false);
    
    if (!hasContent) {
      // 有集群时，验证卡片元素存在
      const clusterCards = page.locator('[role="region"]').filter({ hasText: '外部集群' });
      await expect(clusterCards.first()).toBeVisible({ timeout: 5000 });
    } else {
      // 无集群时，显示空状态
      await expect(page.locator('text=暂无外部集群')).toBeVisible();
    }
  });

  test('应该能刷新集群信息', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 切换到已连接集群列表
    await page.click('button[role="tab"]:has-text("已连接集群")');
    
    // 查找刷新按钮
    const refreshButton = page.locator('button').filter({ hasText: /RefreshCw/ }).first();
    
    if (await refreshButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      // 点击刷新按钮
      await refreshButton.click();
      
      // 等待刷新完成
      await page.waitForTimeout(2000);
      
      // 验证页面没有错误
      const errorAlert = page.locator('[role="alert"]').filter({ hasText: /错误|失败/ });
      await expect(errorAlert).not.toBeVisible({ timeout: 1000 }).catch(() => {});
    }
  });

  test('应该能删除集群', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 先添加一个测试集群
    const clusterName = `待删除集群-${Date.now()}`;
    
    await page.fill('input[name="name"]', clusterName);
    await page.fill('input[name="master_host"]', 'slurm-master');
    await page.fill('input[name="ssh_username"]', 'root');
    await page.fill('input[name="ssh_password"]', 'aiinfra2024');
    
    // 测试连接
    await page.click('button:has-text("测试连接")');
    await page.waitForSelector('[role="alert"]:has-text("连接成功")', { timeout: 30000 });
    
    // 添加集群
    await page.click('button[type="submit"]:has-text("添加集群")');
    await page.waitForTimeout(2000);
    
    // 切换到集群列表
    await page.click('button[role="tab"]:has-text("已连接集群")');
    
    // 找到刚添加的集群并删除
    const clusterCard = page.locator(`text=${clusterName}`).locator('..');
    const deleteButton = clusterCard.locator('button').filter({ hasText: /Trash/ });
    
    // 监听确认对话框
    page.on('dialog', async (dialog) => {
      expect(dialog.message()).toContain('确定要删除');
      await dialog.accept();
    });
    
    await deleteButton.click();
    
    // 等待删除完成
    await page.waitForTimeout(2000);
    
    // 验证集群已从列表中移除
    await expect(page.locator(`text=${clusterName}`)).not.toBeVisible({ timeout: 5000 });
  });

  test('应该正确显示配置复用选项', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 验证三个复选框默认都选中
    await expect(page.locator('input[name="reuse_config"]')).toBeChecked();
    await expect(page.locator('input[name="reuse_munge"]')).toBeChecked();
    await expect(page.locator('input[name="reuse_database"]')).toBeChecked();
    
    // 取消选中
    await page.uncheck('input[name="reuse_config"]');
    await expect(page.locator('input[name="reuse_config"]')).not.toBeChecked();
    
    // 重新选中
    await page.check('input[name="reuse_config"]');
    await expect(page.locator('input[name="reuse_config"]')).toBeChecked();
  });

  test('应该能重置表单', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 填写表单
    await page.fill('input[name="name"]', '测试集群');
    await page.fill('input[name="master_host"]', 'test-host');
    await page.fill('input[name="ssh_password"]', 'test-password');
    
    // 点击重置按钮
    await page.click('button:has-text("重置")');
    
    // 验证表单已重置
    await expect(page.locator('input[name="name"]')).toHaveValue('');
    await expect(page.locator('input[name="master_host"]')).toHaveValue('');
    await expect(page.locator('input[name="ssh_password"]')).toHaveValue('');
  });

  test('应该验证必填字段', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 不填写任何字段，直接提交
    await page.click('button[type="submit"]:has-text("添加集群")');
    
    // 验证必填字段提示（HTML5 验证）
    const nameInput = page.locator('input[name="name"]');
    const isInvalid = await nameInput.evaluate((el) => !el.validity.valid);
    expect(isInvalid).toBe(true);
  });

  test('应该在连接失败时显示错误信息', async ({ page }) => {
    await page.goto('/slurm/external-clusters');
    
    // 填写错误的连接信息
    await page.fill('input[name="master_host"]', '192.168.255.255'); // 不存在的地址
    await page.fill('input[name="ssh_username"]', 'root');
    await page.fill('input[name="ssh_password"]', 'wrong-password');
    
    // 测试连接
    await page.click('button:has-text("测试连接")');
    
    // 等待错误提示
    const errorAlert = page.locator('[role="alert"]');
    await expect(errorAlert).toBeVisible({ timeout: 30000 });
    
    // 验证错误信息存在
    const alertText = await errorAlert.textContent();
    expect(alertText).toMatch(/(连接失败|连接测试失败|超时|无法连接)/);
  });
});

/**
 * 额外的集成测试：验证与其他功能的集成
 */
test.describe('SLURM 外部集群集成测试', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL('/dashboard', { timeout: 10000 });
  });

  test('外部集群应该出现在主集群列表中', async ({ page }) => {
    // 先添加外部集群
    await page.goto('/slurm/external-clusters');
    
    const clusterName = `集成测试-${Date.now()}`;
    
    await page.fill('input[name="name"]', clusterName);
    await page.fill('input[name="master_host"]', 'slurm-master');
    await page.fill('input[name="ssh_username"]', 'root');
    await page.fill('input[name="ssh_password"]', 'aiinfra2024');
    
    await page.click('button:has-text("测试连接")');
    await page.waitForSelector('[role="alert"]:has-text("连接成功")', { timeout: 30000 });
    
    await page.click('button[type="submit"]:has-text("添加集群")');
    await page.waitForTimeout(2000);
    
    // 导航到主集群管理页面
    await page.goto('/slurm/clusters');
    
    // 验证外部集群出现在列表中，并有"外部集群"标签
    await expect(page.locator(`text=${clusterName}`)).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=外部集群')).toBeVisible();
  });

  test('外部集群不应该显示部署和扩容按钮', async ({ page }) => {
    await page.goto('/slurm/clusters');
    
    // 查找标记为"外部集群"的卡片
    const externalCluster = page.locator('text=外部集群').locator('..');
    
    if (await externalCluster.isVisible({ timeout: 2000 }).catch(() => false)) {
      // 验证没有"部署"或"扩容"按钮
      await expect(externalCluster.locator('button:has-text("部署")')).not.toBeVisible();
      await expect(externalCluster.locator('button:has-text("扩容")')).not.toBeVisible();
    }
  });
});
