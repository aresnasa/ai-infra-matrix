# AI Infra Matrix 测试框架

统一的SSO、Gitea、JupyterHub集成测试框架。

## 快速开始

### 安装依赖

```bash
pip install -r requirements.txt
```

### 运行测试

#### 快速验证（推荐）
```bash
python run_tests.py --test quick
```

#### SSO完整测试
```bash
python run_tests.py --test sso -v
```

#### 系统健康检查
```bash
python run_tests.py --test health
```

#### 完整测试套件
```bash
python run_tests.py --test all -v
```

### 自定义URL
```bash
python run_tests.py --url http://your-server:8080 --test sso
```

## 测试模块

### 1. 统一测试框架 (`unified_test.py`)
- 完整的测试框架
- 支持多种测试类型
- 详细的日志输出

### 2. 快速测试 (`quick_test.py`) 
- 最简化的SSO验证
- 适合快速检查

### 3. SSO专项测试 (`sso_tests.py`)
- 专门的SSO功能测试
- 包含所有SSO相关场景

### 4. 工具模块 (`utils.py`)
- 测试会话管理
- 验证器
- 报告器

### 5. 配置文件 (`config.py`)
- 测试配置
- 端点定义
- 测试场景

## 测试场景

### SSO测试
1. **基础认证测试**：验证登录API功能
2. **SSO重定向测试**：验证已登录用户自动重定向
3. **无token访问测试**：验证未登录用户看到登录表单
4. **原始问题验证**：确认已登录用户无需二次密码输入

### 系统健康检查
- 前端服务状态
- 后端API状态  
- Gitea服务状态
- JupyterHub服务状态

## 历史测试文件

以下文件已移动到test文件夹：
- `test_original_problem.py` - 原始问题测试
- `test_sso_complete.py` - 完整SSO测试
- `test_auth_direct.py` - 直接认证测试
- `test_final.py` - 最终验证测试
- `test_gitea_*.py` - Gitea相关测试
- `test_login_experience.py` - 登录体验测试
- `test_user_experience.py` - 用户体验测试
- `diagnose_sso.py` - SSO诊断工具
- `final_verification.py` - 最终验证
- `sso_gitea_test.py` - SSO Gitea测试

## 示例输出

```
[09:53:15] 🚀 快速验证测试
[09:53:15] 🔐 尝试登录用户: admin
[09:53:15] ✅ 登录成功，Token: eyJhbGciOiJIUzI1NiIsI...
[09:53:15] ℹ️ 验证原始问题已解决
[09:53:15] ✅ ✅ 原始问题已解决：已登录用户无需二次密码
[09:53:15] ✅ 快速验证通过
```

## 故障排除

### 连接失败
- 确认服务正在运行：`docker compose ps`
- 检查URL是否正确
- 验证端口是否开放

### 认证失败
- 检查用户名密码是否正确
- 确认后端API正常工作
- 查看后端日志：`docker compose logs backend`

### SSO测试失败
- 检查nginx配置是否正确
- 验证token是否有效
- 查看nginx日志：`docker compose logs nginx`

## 开发指南

### 添加新测试
1. 在相应模块中添加测试方法
2. 更新配置文件中的测试场景
3. 在主运行器中注册新测试

### 扩展测试框架
- 继承`TestSession`类创建专用客户端
- 继承`TestReporter`类自定义输出格式
- 在`config.py`中添加新的测试配置
