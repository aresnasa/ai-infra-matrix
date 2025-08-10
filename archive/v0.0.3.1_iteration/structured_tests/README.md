# AI Infrastructure Matrix 测试套件

本目录包含AI基础设施矩阵项目的完整测试套件，按功能模块组织。

## 目录结构

### 📂 iframe/ - iframe功能测试
专门测试iframe相关功能，包括白屏检测、内容加载验证等。

**主要文件：**
- `test_iframe_auto_login.py` - iframe自动登录综合测试
- `test_iframe_fix_verification.py` - iframe修复验证
- `quick_iframe_test.py` - 快速iframe功能检测
- `iframe_white_screen_fixer.py` - 白屏问题修复工具

### 📂 jupyterhub/ - JupyterHub服务测试  
JupyterHub相关的所有测试，包括配置验证、路由测试、一致性检查等。

**主要文件：**
- `test_jupyterhub_wrapper_optimized.py` - 优化版wrapper测试
- `test_jupyterhub_login_complete.py` - 完整登录流程测试
- `test_jupyterhub_routing_selenium.py` - 路由selenium测试
- `test_jupyterhub_consistency.py` - 一致性测试

### 📂 browser/ - 浏览器测试
浏览器兼容性、缓存行为、自动化测试等。

**主要文件：**
- `test_browser_cache.py` - 浏览器缓存行为测试
- `test_chrome_auto_login.py` - Chrome自动登录测试
- `monitor_chrome_test.py` - Chrome监控测试
- `test_real_browser.py` - 真实浏览器行为测试

### 📂 login/ - 登录认证测试
各种登录场景、SSO认证、自动登录功能测试。

**主要文件：**
- `test_simple_auto_login.py` - 简单自动登录测试
- `test_quick_login.py` - 快速登录测试
- `test_sso_complete.py` - SSO完整流程测试

### 📂 api/ - API和重定向测试
API端点测试、URL重定向验证等。

**主要文件：**
- `test_api_endpoints.py` - API端点测试
- `test_complete_redirect_fix.py` - 重定向修复测试
- `test_js_redirects.py` - JavaScript重定向测试

### 📂 integration/ - 集成测试
完整的端到端测试、集成流程验证。

**主要文件：**
- `test_complete_flow.py` - 完整访问流程测试（包含自动登录）
- `simple_wrapper_test.py` - 简单wrapper集成测试
- `test_final_verification.py` - 最终验证测试

### 📂 utils/ - 测试工具
测试辅助工具、环境检查、验证脚本等。

**主要文件：**
- `check_chrome_env.py` - Chrome环境检查
- `verify_portal_consistency.py` - 门户一致性验证
- `final_verification.py` - 最终验证工具

## 运行测试

### 单个测试文件
```bash
python tests/iframe/quick_iframe_test.py
python tests/integration/test_complete_flow.py
```

### 按模块运行
```bash
# 运行所有iframe测试
python -m pytest tests/iframe/

# 运行所有集成测试  
python -m pytest tests/integration/
```

### 主要测试场景

#### 🎯 iframe白屏问题验证
```bash
python tests/iframe/quick_iframe_test.py
python tests/iframe/test_iframe_fix_verification.py
```

#### 🔐 自动登录功能测试
```bash
python tests/login/test_simple_auto_login.py
python tests/integration/test_complete_flow.py
```

#### 🌐 完整流程验证
```bash
python tests/integration/test_complete_flow.py
python tests/integration/simple_wrapper_test.py
```

## 测试依赖

主要依赖包含在 `requirements-test.txt` 中：
```
selenium
requests
```

确保Chrome浏览器和chromedriver已安装并在PATH中。

## 测试覆盖的功能

✅ iframe白屏问题修复  
✅ 自动登录（admin/admin123）  
✅ JupyterHub wrapper优化  
✅ URL重定向修复  
✅ 浏览器兼容性  
✅ SSO认证流程  
✅ API端点验证  
✅ 完整集成测试  

## 注意事项

1. 大部分测试需要Docker服务运行在 `http://localhost:8080`
2. Chrome相关测试需要安装Chrome浏览器和chromedriver
3. 某些测试可能需要网络连接来验证外部资源
4. 在CI/CD环境中建议使用headless模式运行浏览器测试

## 贡献指南

添加新测试时请：
1. 将测试文件放在合适的目录中
2. 使用描述性的文件名
3. 在文件开头添加清晰的文档注释
4. 包含适当的错误处理和日志记录
5. 更新相关的README文档
