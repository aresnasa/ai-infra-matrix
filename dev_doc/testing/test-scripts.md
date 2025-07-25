# 端到端测试脚本说明

## 概览

`test-e2e.sh` 是一个综合性的端到端测试脚本，集成了所有最近开发和测试的功能，确保 Ansible Playbook Generator 系统的完整性和可靠性。

## 主要功能

### 🔧 已集成的测试功能

1. **服务启动和连接测试**
   - 后端服务健康检查
   - 前端服务可访问性验证
   - 服务间连接测试

2. **用户认证系统测试**
   - 管理员登录验证
   - Token获取和管理
   - 认证状态持久性检查

3. **容器时区配置测试**
   - 验证容器时区设置 (Asia/Shanghai CST)
   - 后端/前端时间同步检查

4. **项目管理功能测试**
   - 动态项目创建
   - 项目配置验证
   - CRUD操作测试

5. **增强健康检查测试** ⭐ 新增
   - 基础健康检查 (`/health`)
   - 数据库连接检查 (`/health/db`)
   - Redis连接检查 (`/health/redis`)
   - API文档可访问性 (`/swagger/index.html`)

6. **用户管理功能测试** ⭐ 新增
   - 用户注册功能
   - 用户登录验证
   - 用户资料管理
   - 权限控制验证 (RBAC)
   - 管理员功能测试

7. **垃圾箱/回收站功能测试** ⭐ 核心功能
   - 软删除 (`PATCH /projects/{id}/soft-delete`)
   - 垃圾箱列表查看 (`GET /projects/trash`)
   - 项目恢复 (`PATCH /projects/{id}/restore`)
   - 永久删除 (`DELETE /projects/{id}/force`)

8. **Playbook相关功能测试**
   - 预览功能 (`POST /playbook/preview`)
   - 包生成 (`POST /playbook/package`)
   - ZIP下载 (`GET /playbook/download-zip/{path}`)
   - 单文件生成 (`POST /playbook/generate`)
   - 单文件下载 (`GET /playbook/download/{id}`)

## 使用方法

### 基本运行
```bash
# 进入测试脚本目录
cd tests/scripts

# 运行完整端到端测试
./test-e2e.sh
```

### 配置选项

测试脚本支持通过 `test-config.env` 文件进行配置：

```bash
# 编辑配置文件
vi test-config.env

# 主要配置项
BASE_URL="http://localhost:8082/api"
FRONTEND_URL="http://localhost:3001"
ENABLE_USER_MANAGEMENT_TESTS=true
ENABLE_HEALTH_CHECK_TESTS=true
ENABLE_TRASH_FUNCTIONALITY_TESTS=true
```

## 测试报告

### 自动报告生成

脚本运行后会自动生成详细的测试报告：
- 位置: `../reports/e2e-test-report-{timestamp}.md`
- 包含详细的测试结果、系统信息和建议

### 测试成功率评估

- **优秀 (90-100%)**: 系统可投入生产
- **良好 (80-89%)**: 需要修复少量问题
- **一般 (70-79%)**: 需要进一步开发
- **较差 (60-69%)**: 不建议投入使用
- **失败 (<60%)**: 需要重新开发

## 测试环境要求

### 前置条件
1. Docker 和 Docker Compose 已安装
2. 服务已启动 (`docker-compose up -d`)
3. 必要的依赖工具：
   - `curl` - API请求
   - `jq` - JSON处理
   - `python3` - URL编码

### 服务端口
- 后端 API: `localhost:8082`
- 前端应用: `localhost:3001`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

## 新增测试覆盖

### 🆕 最近集成的测试用例

1. **用户管理系统**
   - 用户注册/登录流程验证
   - 用户资料管理测试
   - 权限控制系统验证
   - 管理员功能完整性检查

2. **增强健康检查**
   - 多层次健康状态验证
   - 外部依赖连接检查
   - API文档服务验证

3. **回收站功能**
   - 完整的软删除流程
   - 恢复机制验证
   - 永久删除确认

## 故障排除

### 常见问题

1. **服务启动失败**
   ```bash
   # 检查服务状态
   docker-compose ps
   
   # 查看日志
   docker-compose logs
   ```

2. **认证失败**
   ```bash
   # 验证默认管理员账户
   curl -X POST http://localhost:8082/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username":"admin","password":"admin123"}'
   ```

3. **API调用失败**
   ```bash
   # 检查后端健康状态
   curl http://localhost:8082/api/health
   ```

## 维护和更新

### 添加新测试用例

1. 在脚本中添加新的测试函数
2. 在 `main()` 函数中调用新测试
3. 更新测试计数器
4. 添加相应的文档说明

### 配置新的测试环境

1. 修改 `test-config.env` 中的配置
2. 更新容器名称和端口映射
3. 调整测试超时设置

## 总结

这个增强的端到端测试脚本现在提供了：

✅ **13个主要测试模块**
✅ **完整的用户管理测试覆盖**
✅ **增强的健康检查机制**
✅ **全面的垃圾箱功能验证**
✅ **自动化测试报告生成**
✅ **模块化配置管理**
✅ **详细的错误处理和调试信息**

所有之前开发和手动测试的功能现在都已经整合到这个统一的自动化测试脚本中，确保系统的完整性和可靠性。
