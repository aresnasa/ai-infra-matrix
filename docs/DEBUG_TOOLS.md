# AI-Infra-Matrix 调试工具说明

## 概述

AI-Infra-Matrix 支持开发模式和生产模式两种部署方式。在开发模式下，系统提供了丰富的调试工具来帮助开发者诊断和解决问题。

## 🔧 开发模式 vs 生产模式

### 开发模式 (Development Mode)
- **调试工具**: 启用完整的调试工具套件
- **调试路由**: `/debug/` 路径可访问
- **安全性**: 较低的安全限制，便于开发调试
- **性能**: 可能包含额外的日志和调试信息

### 生产模式 (Production Mode)  
- **调试工具**: 完全禁用调试工具
- **调试路由**: `/debug/` 路径返回禁用页面
- **安全性**: 高安全性配置
- **性能**: 优化的性能配置

## 🚀 快速启动

### 方法一：使用构建脚本

```bash
# 开发模式
./scripts/build.sh dev

# 生产模式  
./scripts/build.sh prod

# 开发模式，仅重建nginx
./scripts/build.sh dev --nginx-only

# 生产模式，无缓存构建
./scripts/build.sh prod --no-cache
```

### 方法二：使用智能启动脚本

```bash
# 自动检测环境
./scripts/quick-start.sh

# 强制指定开发模式
./scripts/quick-start.sh dev

# 强制指定生产模式
./scripts/quick-start.sh prod
```

### 方法三：直接使用docker-compose

```bash
# 开发模式
DEBUG_MODE=true BUILD_ENV=development docker-compose --env-file .env.development up -d

# 生产模式
DEBUG_MODE=false BUILD_ENV=production docker-compose --env-file .env.production up -d
```

## 🔍 调试工具套件

### 1. 调试工具首页
- **访问地址**: http://localhost:8080/debug/
- **功能**: 所有调试工具的统一入口
- **包含**: 工具导航、系统状态、快速操作

### 2. JupyterHub 认证调试器
- **访问地址**: http://localhost:8080/debug/debug_jupyterhub_auth.html
- **功能**: 
  - 实时认证流程监控
  - 详细的API调用日志
  - Token验证测试
  - 错误诊断

### 3. Token 管理工具
- **访问地址**: http://localhost:8080/debug/token_setup.html
- **功能**:
  - Token设置和管理
  - Mock Token生成
  - Token有效性验证
  - 存储状态检查

### 4. 认证流程测试
- **访问地址**: http://localhost:8080/debug/test_jupyterhub_auth.html
- **功能**:
  - 各种认证场景测试
  - iframe集成测试
  - 错误处理测试

## 📁 项目结构

```
src/
├── shared/
│   ├── debug/                    # 调试工具目录
│   │   ├── index.html           # 调试工具首页
│   │   ├── debug_jupyterhub_auth.html
│   │   ├── token_setup.html
│   │   └── test_jupyterhub_auth.html
│   ├── sso/                     # SSO相关页面
│   └── jupyterhub/              # JupyterHub相关页面
├── nginx/
│   ├── Dockerfile               # 支持DEBUG_MODE参数
│   ├── nginx.conf               # 包含调试路由配置
│   └── docker-entrypoint.sh     # 动态配置处理
└── ...
```

## ⚙️ 环境配置

### .env.development
```bash
DEBUG_MODE=true
BUILD_ENV=development
# 其他开发环境配置...
```

### .env.production
```bash
DEBUG_MODE=false
BUILD_ENV=production
# 其他生产环境配置...
```

## 🔒 安全考虑

### 开发模式安全提示
- 调试工具包含敏感信息显示
- 不应在公网环境中启用
- 建议仅在本地开发环境使用

### 生产模式安全保证
- 完全禁用调试工具
- 移除调试路由
- 优化的安全配置

## 🐛 故障排除

### 常见问题

1. **调试工具无法访问**
   - 检查是否使用开发模式构建
   - 确认 `DEBUG_MODE=true`
   - 重新构建nginx服务: `./scripts/build.sh dev --nginx-only`

2. **Token验证失败**
   - 使用Token管理工具检查Token状态
   - 确认后端服务运行正常
   - 检查API端点连通性

3. **认证流程异常**
   - 使用认证调试器查看详细日志
   - 检查浏览器控制台错误
   - 验证localStorage中的Token

### 日志查看

```bash
# 查看nginx日志
docker-compose logs -f nginx

# 查看所有服务状态
docker-compose ps

# 健康检查
curl http://localhost:8080/health
```

## 📊 监控和诊断

### 系统健康检查
- **端点**: http://localhost:8080/health
- **用途**: 验证nginx服务状态

### API连接测试
- **端点**: http://localhost:8080/api/auth/verify
- **用途**: 测试后端API连通性

### 调试信息收集
- 浏览器开发者工具Console
- Network面板API调用记录
- 调试工具页面的实时日志

## 🔄 切换模式

### 从开发模式切换到生产模式
```bash
./scripts/build.sh prod --rebuild
```

### 从生产模式切换到开发模式
```bash
./scripts/build.sh dev --rebuild
```

### 验证当前模式
- 访问 http://localhost:8080/debug/
- 开发模式: 显示调试工具界面
- 生产模式: 显示禁用提示页面

## 🤝 贡献指南

在开发新功能或修复问题时：

1. 使用开发模式进行测试
2. 充分利用调试工具进行诊断
3. 确保生产模式下调试工具被正确禁用
4. 更新相关文档

## 📞 技术支持

如需技术支持，请提供：
- 当前运行模式 (开发/生产)
- 错误信息截图
- 相关日志输出
- 浏览器控制台错误
