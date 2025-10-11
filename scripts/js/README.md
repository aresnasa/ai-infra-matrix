# Playwright MCP 测试套件

这个目录包含使用Playwright和playwright-mcp进行的UI自动化测试。

## 📁 文件说明

### 测试文件

1. **test-login-playwright.js** - 登录和认证测试
   - 测试主页访问
   - 测试用户登录（UI和API）
   - 测试认证状态检查
   - 测试JupyterHub wrapper页面
   - 测试iframe元素
   - 测试localStorage token
   - 测试API健康检查
   - 测试Projects页面访问
   - 测试JavaScript控制台错误

2. **test-mcp-runner.js** - MCP测试运行器
   - 演示playwright-mcp的使用
   - 自动化测试流程
   - 生成详细测试报告

3. **test-page.html** - 测试页面
   - 独立的登录测试页面
   - 演示前端功能
   - localStorage管理
   - iframe集成示例

### 其他测试文件

- **test-object-storage-playwright.js** - 对象存储测试
- **test-saltstack-playwright.js** - SaltStack测试
- **test-salt-execute-playwright.js** - Salt执行测试
- **test-slurm-saltstack.js** - Slurm SaltStack测试
- **test-slurm-scaleup-playwright.js** - Slurm扩展测试

## 🚀 安装

在scripts目录下安装依赖：

```bash
npm install
```

安装Playwright浏览器：

```bash
npm run install-browsers
```

## 🧪 运行测试

### 登录和认证测试

```bash
# 无头模式（CI友好）
npm run test:login:ci

# 有头模式（可视化调试）
npm run test:login:headed

# 默认运行
npm run test:login
```

### MCP测试运行器

```bash
# 无头模式
npm run test:mcp

# 有头模式
npm run test:mcp:headed
```

### 其他测试

```bash
# 对象存储测试
npm run test

# SaltStack测试
npm run test:salt

# Slurm测试
npm run test:slurm
```

## 🌐 测试页面

在scripts/js目录下启动HTTP服务器：

```bash
cd scripts/js
python3 -m http.server 8080
```

然后在浏览器中访问：
- http://localhost:8080/test-page.html

## 📝 使用playwright-mcp进行测试

playwright-mcp提供了一个MCP服务器，可以通过工具调用来控制浏览器。以下是示例：

### 1. 导航到页面

```javascript
playwright-browser_navigate({
    url: "http://localhost:8080/test-page.html"
})
```

### 2. 获取页面快照

```javascript
playwright-browser_snapshot()
```

### 3. 点击元素

```javascript
playwright-browser_click({
    element: "Login button",
    ref: "e11"  // 从快照中获取
})
```

### 4. 填写表单

```javascript
playwright-browser_fill_form({
    fields: [
        {
            name: "username",
            type: "textbox",
            ref: "e7",
            value: "admin"
        },
        {
            name: "password",
            type: "textbox",
            ref: "e10",
            value: "admin123"
        }
    ]
})
```

### 5. 执行JavaScript

```javascript
playwright-browser_evaluate({
    function: "() => { return localStorage.getItem('token'); }"
})
```

### 6. 截图

```javascript
playwright-browser_take_screenshot({
    filename: "test-result.png"
})
```

## 🎯 测试覆盖范围

### 登录和认证测试覆盖：

- ✅ 主页加载
- ✅ API健康检查
- ✅ API登录
- ✅ 认证状态验证
- ✅ localStorage token验证
- ✅ JupyterHub wrapper页面
- ✅ iframe元素检查
- ✅ Projects页面访问
- ✅ JavaScript控制台错误检查

### MCP测试运行器覆盖：

- ✅ 页面加载
- ✅ 表单元素存在性
- ✅ 登录功能
- ✅ localStorage验证
- ✅ 用户信息显示
- ✅ iframe元素检查
- ✅ 认证检查功能
- ✅ 登出功能
- ✅ 控制台错误检查

## ⚙️ 环境变量

可以通过环境变量配置测试：

- `FRONTEND_URL` - 前端基础URL（默认: http://localhost:8080）
- `HEADLESS` - 无头模式（默认: true）
- `SCREENSHOT` - 是否保存截图（默认: true）
- `SCREENSHOT_PATH` - 截图保存路径（默认: ./test-screenshots）
- `TIMEOUT` - 操作超时时间（默认: 30000ms）
- `SLOWMO` - 慢速模式延迟（默认: 0ms）
- `TEST_USERNAME` - 测试用户名（默认: admin）
- `TEST_PASSWORD` - 测试密码（默认: admin123）

## 📊 测试报告

测试完成后会生成详细的报告，包括：

- 测试统计（总数、通过、失败、通过率）
- 每个测试的详细结果
- 截图保存路径
- 测试配置信息

示例输出：

```
======================================================================
🧪 AI Infrastructure Matrix - 登录和认证测试报告
======================================================================
📊 测试统计:
   总计: 9
   通过: 9 ✅
   失败: 0 ❌
   通过率: 100.00%

📋 详细结果:
   1. ✅ homepage_access: 主页加载成功 (HTTP 200)
   2. ✅ api_health_check: API健康状态正常 (200)
   3. ✅ api_login: API登录成功，token已设置
   4. ✅ auth_status_check: 认证状态有效，用户: admin
   5. ✅ token_storage_check: Token存在于localStorage，长度: 125
   6. ✅ jupyterhub_wrapper: JupyterHub wrapper页面加载成功
   7. ✅ iframe_element_check: 找到1个iframe元素，src: about:blank
   8. ✅ projects_page_access: Projects页面访问成功
   9. ✅ console_errors_check: 未发现控制台错误

🖼️  截图保存路径: ./test-screenshots
🌐 测试基础URL: http://localhost:8080
👤 测试账户: admin
======================================================================
```

## 🔍 故障排查

### 测试失败

1. 检查应用是否正在运行
2. 验证URL配置是否正确
3. 查看截图文件了解失败时的页面状态
4. 检查控制台输出的详细错误信息

### 浏览器未安装

运行以下命令安装Playwright浏览器：

```bash
npx playwright install
```

### 网络问题

如果遇到网络问题，可以：

1. 配置npm代理
2. 使用淘宝npm镜像
3. 在有网络的环境中预先安装依赖

## 📚 参考资料

- [Playwright官方文档](https://playwright.dev/)
- [MCP协议文档](https://modelcontextprotocol.io/)
- [AI Infrastructure Matrix文档](../../README.md)

## 🤝 贡献

欢迎提交PR改进测试套件！

## 📄 许可证

MIT
