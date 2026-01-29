// @ts-check
/**
 * 邀请码注册功能测试
 * 
 * 测试场景：
 * 1. 邀请码验证 API
 * 2. 使用有效邀请码注册（即时激活）
 * 3. 不使用邀请码注册（需要管理员审批）
 * 4. 使用无效邀请码注册（应显示错误）
 */
const { test, expect } = require('@playwright/test');

// 使用 HTTPS 端口
const BASE_URL = process.env.BASE_URL || 'https://localhost:8443';
const ADMIN_USER = process.env.ADMIN_USER || 'admin';
const ADMIN_PASS = process.env.ADMIN_PASS || 'admin123';

// 测试用户配置
const TEST_USER_WITH_CODE = {
  username: `testuser_code_${Date.now()}`,
  email: `testuser_code_${Date.now()}@example.com`,
  password: 'TestPass123!',
  department: 'engineering',
  roleTemplate: 'data-developer',
};

const TEST_USER_WITHOUT_CODE = {
  username: `testuser_nocode_${Date.now()}`,
  email: `testuser_nocode_${Date.now()}@example.com`,
  password: 'TestPass123!',
  department: 'engineering',
  roleTemplate: 'data-developer',
};

// Helper: 管理员登录获取 token
async function adminLogin(request) {
  const res = await request.post(`${BASE_URL}/api/auth/login`, {
    data: {
      username: ADMIN_USER,
      password: ADMIN_PASS,
    },
    ignoreHTTPSErrors: true,
  });
  
  if (res.status() !== 200) {
    const text = await res.text();
    throw new Error(`Admin login failed: ${res.status()} ${text}`);
  }
  
  const data = await res.json();
  return data.token;
}

// Helper: 创建邀请码
async function createInvitationCode(request, token, options: any = {}) {
  const res = await request.post(`${BASE_URL}/api/admin/invitation-codes`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    data: {
      description: options.description || 'E2E Test Invitation Code',
      role_template: options.roleTemplate || 'data-developer',
      max_uses: options.maxUses || 1,
      expires_in_days: options.expiresInDays || 7,
    },
    ignoreHTTPSErrors: true,
  });
  
  if (res.status() !== 201) {
    const text = await res.text();
    throw new Error(`Create invitation code failed: ${res.status()} ${text}`);
  }
  
  const data = await res.json();
  // API 返回格式是 { code: { id, code, ... }, message: "..." }
  return data.code;
}

// Helper: 删除邀请码
async function deleteInvitationCode(request, token, codeId) {
  await request.delete(`${BASE_URL}/api/admin/invitation-codes/${codeId}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    ignoreHTTPSErrors: true,
  });
}

// Helper: 删除用户（清理）
async function deleteUser(request, token, userId) {
  await request.delete(`${BASE_URL}/api/admin/users/${userId}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    ignoreHTTPSErrors: true,
  });
}

test.describe('邀请码注册功能测试', () => {
  let adminToken: string;
  let createdInvitationCodeId: number;
  let createdUserId: number;

  test.beforeAll(async ({ request }) => {
    // 获取管理员 token
    adminToken = await adminLogin(request);
  });

  test.afterAll(async ({ request }) => {
    // 清理创建的邀请码
    if (createdInvitationCodeId) {
      try {
        await deleteInvitationCode(request, adminToken, createdInvitationCodeId);
      } catch (e) {
        console.log('Cleanup invitation code skipped:', e);
      }
    }
    // 清理创建的用户
    if (createdUserId) {
      try {
        await deleteUser(request, adminToken, createdUserId);
      } catch (e) {
        console.log('Cleanup user skipped:', e);
      }
    }
  });

  test('1. API: 创建邀请码', async ({ request }) => {
    const result = await createInvitationCode(request, adminToken, {
      description: 'E2E Test Code',
      roleTemplate: 'data-developer',
      maxUses: 5,
      expiresInDays: 1,
    });

    expect(result.code).toBeDefined();
    expect(result.code.length).toBeGreaterThan(10);
    expect(result.description).toBe('E2E Test Code');
    expect(result.max_uses).toBe(5);
    
    createdInvitationCodeId = result.id;
    console.log(`Created invitation code: ${result.code}`);
  });

  test('2. API: 验证有效邀请码', async ({ request }) => {
    // 先创建一个邀请码
    const invCode = await createInvitationCode(request, adminToken, {
      description: 'Validation Test',
      maxUses: 1,
    });

    // 验证邀请码
    const res = await request.get(`${BASE_URL}/api/auth/validate-invitation-code?code=${invCode.code}`);
    expect(res.status()).toBe(200);
    
    const data = await res.json();
    expect(data.valid).toBe(true);
    expect(data.role_template).toBe('data-developer');

    // 清理
    await deleteInvitationCode(request, adminToken, invCode.id);
  });

  test('3. API: 验证无效邀请码', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/auth/validate-invitation-code?code=INVALID_CODE_12345`);
    expect(res.status()).toBe(200);
    
    const data = await res.json();
    expect(data.valid).toBe(false);
  });

  test('4. API: 使用邀请码注册（直接激活）', async ({ request }) => {
    // 创建邀请码
    const invCode = await createInvitationCode(request, adminToken, {
      description: 'Registration Test',
      roleTemplate: 'data-developer',
      maxUses: 1,
    });

    // 使用邀请码注册
    const res = await request.post(`${BASE_URL}/api/auth/register`, {
      data: {
        username: TEST_USER_WITH_CODE.username,
        email: TEST_USER_WITH_CODE.email,
        password: TEST_USER_WITH_CODE.password,
        department: TEST_USER_WITH_CODE.department,
        role_template: TEST_USER_WITH_CODE.roleTemplate,
        invitation_code: invCode.code,
      },
    });

    console.log(`Register with code response: ${res.status()}`);
    const data = await res.json();
    console.log(`Register response:`, JSON.stringify(data, null, 2));

    // 使用邀请码注册应该返回 201 并且用户直接激活
    expect(res.status()).toBe(201);
    expect(data.user).toBeDefined();
    expect(data.user.is_active).toBe(true);
    expect(data.message).toContain('成功');

    createdUserId = data.user.id;

    // 清理邀请码
    await deleteInvitationCode(request, adminToken, invCode.id);

    // 验证可以登录
    const loginRes = await request.post(`${BASE_URL}/api/auth/login`, {
      data: {
        username: TEST_USER_WITH_CODE.username,
        password: TEST_USER_WITH_CODE.password,
      },
    });
    expect(loginRes.status()).toBe(200);
  });

  test('5. API: 不使用邀请码注册（E2E模式下直接创建）', async ({ request }) => {
    // 注意：在 E2E_ALLOW_FAKE_LDAP=true 环境下，不使用邀请码注册会走 bypass 路径直接创建用户
    // 这是为了支持其他 E2E 测试需要快速创建用户的场景
    // 生产环境下（E2E_ALLOW_FAKE_LDAP=false），不使用邀请码注册会返回 202 需要审批
    const res = await request.post(`${BASE_URL}/api/auth/register`, {
      data: {
        username: TEST_USER_WITHOUT_CODE.username,
        email: TEST_USER_WITHOUT_CODE.email,
        password: TEST_USER_WITHOUT_CODE.password,
        department: TEST_USER_WITHOUT_CODE.department,
        role_template: TEST_USER_WITHOUT_CODE.roleTemplate,
        // 不提供 invitation_code
      },
    });

    console.log(`Register without code response: ${res.status()}`);
    const data = await res.json();
    console.log(`Register response:`, JSON.stringify(data, null, 2));

    // E2E 模式下会返回 201 并直接创建用户
    // 非 E2E 模式下应该返回 202（待审批）
    expect([201, 202]).toContain(res.status());
    
    if (res.status() === 202) {
      // 非 E2E 模式：验证审批流程
      expect(data.message).toContain('审批');
      expect(data.requires_approval).toBe(true);
      
      // 尝试登录应该失败（未激活）
      const loginRes = await request.post(`${BASE_URL}/api/auth/login`, {
        data: {
          username: TEST_USER_WITHOUT_CODE.username,
          password: TEST_USER_WITHOUT_CODE.password,
        },
      });
      expect([401, 403]).toContain(loginRes.status());
    } else {
      // E2E 模式：用户直接创建
      expect(data.is_active).toBe(true);
    }
  });

  test('6. UI: 注册页面显示邀请码输入框', async ({ page }) => {
    await page.goto('/');
    
    // 点击注册标签
    await page.getByRole('tab', { name: /注册|Register/i }).click();
    
    // 等待注册表单加载
    await page.waitForSelector('input[placeholder*="邀请码"], input[placeholder*="Invitation"]', { 
      timeout: 10000 
    });
    
    // 验证邀请码输入框存在
    const invitationCodeInput = page.locator('input[placeholder*="邀请码"], input[placeholder*="Invitation"]');
    await expect(invitationCodeInput).toBeVisible();
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/invitation-code-input.png' });
  });

  test('7. UI: 输入有效邀请码显示验证成功', async ({ page, request }) => {
    // 创建邀请码
    const invCode = await createInvitationCode(request, adminToken, {
      description: 'UI Validation Test',
      maxUses: 10,
    });

    await page.goto('/');
    
    // 点击注册标签
    await page.getByRole('tab', { name: /注册|Register/i }).click();
    
    // 等待注册表单加载
    await page.waitForSelector('input[placeholder*="邀请码"], input[placeholder*="Invitation"]', { 
      timeout: 10000 
    });
    
    // 输入邀请码
    const invitationCodeInput = page.locator('input[placeholder*="邀请码"], input[placeholder*="Invitation"]');
    await invitationCodeInput.fill(invCode.code);
    
    // 等待验证结果（防抖 500ms + API 响应）
    await page.waitForTimeout(1500);
    
    // 验证显示成功提示 - 检查多种可能的成功指示
    const successIndicators = [
      page.locator('.ant-alert-success'),
      page.locator('.ant-input-affix-wrapper-status-success'),
      page.locator('[class*="success"]'),
      page.locator('text=有效'),
      page.locator('text=valid'),
    ];
    
    let found = false;
    for (const indicator of successIndicators) {
      if (await indicator.first().isVisible({ timeout: 1000 }).catch(() => false)) {
        found = true;
        break;
      }
    }
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/invitation-code-valid.png' });
    
    expect(found).toBe(true);

    // 清理
    await deleteInvitationCode(request, adminToken, invCode.id);
  });

  test('8. UI: 输入无效邀请码显示错误', async ({ page }) => {
    await page.goto('/');
    
    // 点击注册标签
    await page.getByRole('tab', { name: /注册|Register/i }).click();
    
    // 等待注册表单加载
    await page.waitForSelector('input[placeholder*="邀请码"], input[placeholder*="Invitation"]', { 
      timeout: 10000 
    });
    
    // 输入无效邀请码
    const invitationCodeInput = page.locator('input[placeholder*="邀请码"], input[placeholder*="Invitation"]');
    await invitationCodeInput.fill('INVALID_CODE_XYZ123');
    
    // 等待验证结果
    await page.waitForTimeout(1500);
    
    // 验证显示错误提示
    const errorIndicators = [
      page.locator('.ant-alert-warning'),
      page.locator('.ant-input-affix-wrapper-status-error'),
      page.locator('[class*="error"]'),
      page.locator('[class*="warning"]'),
      page.locator('text=无效'),
      page.locator('text=invalid'),
    ];
    
    let found = false;
    for (const indicator of errorIndicators) {
      if (await indicator.first().isVisible({ timeout: 1000 }).catch(() => false)) {
        found = true;
        break;
      }
    }
    
    // 截图
    await page.screenshot({ path: 'test-screenshots/invitation-code-invalid.png' });
    
    expect(found).toBe(true);
  });

  test('9. UI: 完整注册流程（使用邀请码）', async ({ page, request }) => {
    // 创建邀请码
    const invCode = await createInvitationCode(request, adminToken, {
      description: 'Full Registration Test',
      roleTemplate: 'data-developer',
      maxUses: 1,
    });

    const testUser = {
      username: `uitest_${Date.now()}`,
      email: `uitest_${Date.now()}@test.com`,
      password: 'TestPass123!',
    };

    await page.goto('/');
    
    // 点击注册标签
    await page.getByRole('tab', { name: /注册|Register/i }).click();
    await page.waitForTimeout(500);
    
    // 使用更精确的选择器 - 注册表单的输入框
    const registerForm = page.locator('[role="tabpanel"]').filter({ has: page.locator('input[id*="register"]') });
    
    // 填写用户名 - 使用 ID 选择器
    await page.locator('#register_username, input[id*="register"][placeholder*="用户名"]').fill(testUser.username);
    
    // 填写邮箱
    await page.locator('#register_email, input[id*="register"][placeholder*="邮箱"]').fill(testUser.email);
    
    // 密码字段 - 注册表单的密码字段
    const passwordFields = page.locator('input[type="password"][id*="register"], [role="tabpanel"]:has(input[id*="register"]) input[type="password"]');
    const passwordCount = await passwordFields.count();
    if (passwordCount >= 2) {
      await passwordFields.nth(0).fill(testUser.password);
      await passwordFields.nth(1).fill(testUser.password);
    } else {
      // 备选方案：所有密码字段
      const allPasswordFields = page.locator('input[type="password"]');
      await allPasswordFields.nth(0).fill(testUser.password);
      if (await allPasswordFields.count() > 1) {
        await allPasswordFields.nth(1).fill(testUser.password);
      }
    }
    
    // 填写部门
    const deptInput = page.locator('input[placeholder*="部门"], input[placeholder*="Department"]');
    if (await deptInput.isVisible()) {
      await deptInput.fill('engineering');
    }
    
    // 输入邀请码
    const invitationCodeInput = page.locator('input[placeholder*="邀请码"], input[placeholder*="Invitation"]');
    await invitationCodeInput.fill(invCode.code);
    
    // 等待邀请码验证
    await page.waitForTimeout(1500);
    
    // 截图注册表单填写完成状态
    await page.screenshot({ path: 'test-screenshots/registration-form-filled.png' });
    
    // 点击注册按钮
    const registerButton = page.getByRole('button', { name: /注册|Register|提交/i });
    await registerButton.click();
    
    // 等待注册结果
    await page.waitForTimeout(3000);
    
    // 截图结果
    await page.screenshot({ path: 'test-screenshots/registration-result.png' });
    
    // 验证注册成功提示或跳转到登录
    const successMessage = page.locator('.ant-message-success, .ant-notification-success, [class*="success"]');
    const loginTab = page.getByRole('tab', { name: /登录|Login/i });
    
    // 注册成功后应该显示成功消息或自动切换到登录标签
    const hasSuccess = await successMessage.first().isVisible().catch(() => false);
    const isLoginVisible = await loginTab.isVisible().catch(() => false);
    
    expect(hasSuccess || isLoginVisible).toBe(true);

    // 清理
    await deleteInvitationCode(request, adminToken, invCode.id);
    
    // 如果注册成功，尝试删除用户
    try {
      const usersRes = await request.get(`${BASE_URL}/api/admin/users`, {
        headers: { 'Authorization': `Bearer ${adminToken}` },
        ignoreHTTPSErrors: true,
      });
      const usersData = await usersRes.json();
      // API 返回格式: { users: [...], total, page, ... }
      const usersList = usersData.users || [];
      const createdUser = usersList.find((u: any) => u.username === testUser.username);
      if (createdUser) {
        await deleteUser(request, adminToken, createdUser.id);
      }
    } catch (e) {
      console.log('User cleanup skipped:', e);
    }
  });
});

test.describe('邀请码管理 API 测试', () => {
  let adminToken: string;
  let testCodeId: number;

  test.beforeAll(async ({ request }) => {
    adminToken = await adminLogin(request);
  });

  test.afterAll(async ({ request }) => {
    if (testCodeId) {
      try {
        await deleteInvitationCode(request, adminToken, testCodeId);
      } catch (e) {
        console.log('Cleanup skipped:', e);
      }
    }
  });

  test('列出邀请码', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/admin/invitation-codes`, {
      headers: { 'Authorization': `Bearer ${adminToken}` },
      ignoreHTTPSErrors: true,
    });
    
    expect(res.status()).toBe(200);
    const data = await res.json();
    // API 返回格式: { data: [...], total: number, page: number, page_size: number }
    expect(Array.isArray(data.data)).toBe(true);
    expect(typeof data.total).toBe('number');
    expect(typeof data.page).toBe('number');
  });

  test('创建和禁用邀请码', async ({ request }) => {
    // 创建
    const createRes = await request.post(`${BASE_URL}/api/admin/invitation-codes`, {
      headers: {
        'Authorization': `Bearer ${adminToken}`,
        'Content-Type': 'application/json',
      },
      data: {
        description: 'Disable Test',
        role_template: 'data-developer',
        max_uses: 10,
        expires_in_days: 1,
      },
      ignoreHTTPSErrors: true,
    });
    
    expect(createRes.status()).toBe(201);
    const createData = await createRes.json();
    // API 返回格式: { code: { id, code, ... }, message: "..." }
    const created = createData.code;
    testCodeId = created.id;

    // 禁用 - POST /api/admin/invitation-codes/:id/disable
    const disableRes = await request.post(`${BASE_URL}/api/admin/invitation-codes/${testCodeId}/disable`, {
      headers: { 'Authorization': `Bearer ${adminToken}` },
      ignoreHTTPSErrors: true,
    });
    expect(disableRes.status()).toBe(200);

    // 验证禁用后的邀请码不可用
    const validateRes = await request.get(`${BASE_URL}/api/auth/validate-invitation-code?code=${created.code}`, {
      ignoreHTTPSErrors: true,
    });
    const validateData = await validateRes.json();
    expect(validateData.valid).toBe(false);

    // 重新启用 - POST /api/admin/invitation-codes/:id/enable
    const enableRes = await request.post(`${BASE_URL}/api/admin/invitation-codes/${testCodeId}/enable`, {
      headers: { 'Authorization': `Bearer ${adminToken}` },
      ignoreHTTPSErrors: true,
    });
    expect(enableRes.status()).toBe(200);

    // 验证启用后可用
    const validateRes2 = await request.get(`${BASE_URL}/api/auth/validate-invitation-code?code=${created.code}`, {
      ignoreHTTPSErrors: true,
    });
    const validateData2 = await validateRes2.json();
    expect(validateData2.valid).toBe(true);
  });

  test('获取邀请码统计', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/admin/invitation-codes/statistics`, {
      headers: { 'Authorization': `Bearer ${adminToken}` },
      ignoreHTTPSErrors: true,
    });
    
    expect(res.status()).toBe(200);
    const data = await res.json();
    // API 返回格式: { total_codes: number, active_codes: number, total_usages: number }
    expect(typeof data.total_codes).toBe('number');
    expect(typeof data.active_codes).toBe('number');
    expect(typeof data.total_usages).toBe('number');
  });

  test('创建多个邀请码', async ({ request }) => {
    // 后端不支持批量创建，改为创建3个单独的邀请码来验证
    const createdCodes: any[] = [];
    
    for (let i = 0; i < 3; i++) {
      const res = await request.post(`${BASE_URL}/api/admin/invitation-codes`, {
        headers: {
          'Authorization': `Bearer ${adminToken}`,
          'Content-Type': 'application/json',
        },
        data: {
          description: `Multi Test ${i + 1}`,
          role_template: 'data-developer',
          max_uses: 1,
          expires_in_days: 1,
        },
        ignoreHTTPSErrors: true,
      });
      
      expect(res.status()).toBe(201);
      const data = await res.json();
      createdCodes.push(data.code);
    }
    
    expect(createdCodes.length).toBe(3);

    // 清理
    for (const code of createdCodes) {
      await deleteInvitationCode(request, adminToken, code.id);
    }
  });
});
