# 测试脚本使用指南

本目录包含 Ansible Playbook Generator 项目的所有测试脚本。

## 测试脚本概览

### 主要测试脚本

1. **test-e2e.sh** - 端到端功能测试
   - 健康检查
   - 用户认证
   - Playbook生成
   - ZIP包下载
   - 单文件下载
   - 清理测试

2. **test-api.sh** - API功能测试
   - 基础API端点测试
   - 认证流程测试
   - 项目管理测试

3. **test-health-checks.sh** - 健康检查脚本
   - 容器状态检查
   - 服务健康状态检查
   - 数据库连接测试

4. **test_user_management.sh** - 用户管理测试
   - 用户创建和删除
   - 角色分配测试
   - 权限验证测试

5. **docker-test.sh** - Docker环境测试
   - 容器构建测试
   - 服务启动测试

### 测试数据文件

- **test_single_download.yml** - 单文件下载测试的示例文件
- **test_download.zip** - ZIP下载测试文件（将被清理）
- **test_download_correct.zip** - 修复后的ZIP测试文件（将被清理）

### Go测试文件

- **test_password.go** - 密码哈希测试
- **test_zip_fix.go** - ZIP修复功能测试（将被清理）

## 使用方法

### 运行完整测试套件

```bash
# 运行端到端测试（推荐）
./test-e2e.sh

# 运行API测试
./test-api.sh

# 运行健康检查
./test-health-checks.sh

# 运行用户管理测试
./test_user_management.sh
```

### 运行特定测试

```bash
# 只运行健康检查
curl http://localhost:8082/api/health

# 只测试认证
TOKEN=$(curl -s -X POST http://localhost:8082/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

# 只测试下载功能（需要先生成playbook）
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8082/api/playbook/download/1/playbook.yml" \
  -o test_download.yml
```

## 测试环境要求

### 前置条件

1. Docker和Docker Compose已安装
2. 所有服务容器正常运行
3. 数据库已初始化
4. 具有curl、jq等工具

### 环境变量

测试脚本使用以下默认配置：
- **BACKEND_URL**: http://localhost:8082
- **FRONTEND_URL**: http://localhost:3001
- **DEFAULT_USER**: admin
- **DEFAULT_PASS**: admin123

## 测试结果解读

### 成功标志

- **端到端测试**: 6/6 测试通过
- **HTTP状态码**: 200, 201 表示成功
- **文件大小**: ZIP包约3755字节
- **容器状态**: 所有容器健康运行

### 常见失败原因

1. **服务未启动**: 先运行 `docker-compose up -d`
2. **数据库未初始化**: 运行 `cd backend && ./db_manager.sh init`
3. **端口被占用**: 检查端口3001, 8082, 5433, 6379
4. **权限问题**: 确保测试脚本有执行权限

## 测试维护

### 添加新测试

1. 创建新的测试脚本
2. 遵循现有命名约定
3. 添加适当的错误处理
4. 更新此README文档

### 清理测试数据

```bash
# 清理测试生成的文件
rm -f test_*.yml test_*.zip test_*.json

# 重置测试数据库（谨慎使用）
cd backend && ./db_manager.sh reset && ./db_manager.sh init
```

### 持续集成

测试脚本设计为可以在CI/CD环境中运行：
- 返回适当的退出码
- 生成结构化输出
- 支持静默模式
- 自动清理测试数据
