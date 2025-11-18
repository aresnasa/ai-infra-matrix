# CHANGELOG - 环境变量归一化

## [2024-01-XX] - 环境变量归一化完成

### 🎯 重大变更

#### 环境变量管理重构
- **归一化配置**: 所有环境变量统一到项目根目录 `.env` 文件
- **移除组件级配置**: 删除 `src/*/. env` 文件，避免配置分散
- **统一管理**: 通过 docker-compose 的 `env_file` 机制统一传递

### ✨ 新增功能

#### SaltStack 配置灵活化
- 支持通过环境变量配置 Salt API 连接
- 新增环境变量：
  - `SALTSTACK_MASTER_URL` - 完整 URL 配置
  - `SALT_API_SCHEME` - 协议配置 (http/https)
  - `SALT_MASTER_HOST` - 主机名
  - `SALT_API_PORT` - 端口号
  - `SALT_API_TIMEOUT` - 超时时间
  - `SALT_API_USERNAME` - 用户名
  - `SALT_API_PASSWORD` - 密码
  - `SALT_API_EAUTH` - 认证方式

#### React 应用配置
- 在 `.env.example` 中添加 React 构建时环境变量
- 通过 docker-compose build args 传递配置
- 变量包括：
  - `REACT_APP_API_URL`
  - `REACT_APP_JUPYTERHUB_URL`
  - `REACT_APP_JUPYTERHUB_UNIFIED_AUTH`
  - `REACT_APP_DEBUG_MODE`
  - `REACT_APP_NAME`
  - `REACT_APP_VERSION`

### 🔧 改进

#### 构建脚本优化
- **build.sh**: 移除创建 `src/backend/.env` 的逻辑
- 添加注释说明统一配置方式

#### Dockerfile 清理
- **src/backend/Dockerfile**: 移除 .env 文件复制操作
- 依赖 docker-compose 传递环境变量

#### 配置模板更新
- **.env.example**: 修正 SaltStack 配置（端口 8000，file eauth）
- **src/backend/.env.example**: 添加废弃警告

### 🛠️ 工具和脚本

#### 新增脚本
1. **scripts/migrate-backend-env.sh**
   - 自动化迁移脚本
   - 从 `src/backend/.env` 迁移到 `/.env`
   - 支持备份、冲突检测、交互式确认

2. **scripts/verify-env-cleanup.sh**
   - 环境变量归一化验证脚本
   - 全面检查配置正确性
   - 输出详细验证报告

### 📝 文档

#### 新增文档
1. **src/backend/README_ENV.md**
   - Backend 环境变量配置完整说明
   - 包含配置方法、迁移指南、常见问题

2. **docs/SALTSTACK_INTEGRATION_ENV_CONFIG.md**
   - SaltStack 集成环境变量配置文档
   - 详细的配置项说明和使用示例

3. **docs/ENV_CLEANUP_REPORT.md**
   - 环境变量归一化完成报告
   - 详细记录所有变更和影响

### 🐛 Bug 修复

#### SaltStack 状态显示
- **问题**: /slurm 页面显示 SaltStack 状态为"未知"
- **原因**: 代码返回 demo 数据而非真实 API 状态
- **解决**: 添加 `getRealSaltStackStatus()` 方法直接调用 Salt API

#### 配置硬编码
- **问题**: Salt API 连接信息硬编码在代码中
- **解决**: 改为从环境变量读取，支持灵活配置

### 🗑️ 删除

#### 组件级配置文件
- `src/frontend/.env` - 已删除
- `src/backend/.env` - 已删除

### ⚠️ 破坏性变更

#### 配置文件位置变更
- **之前**: 配置分散在 `src/*/. env` 文件中
- **现在**: 所有配置统一在项目根目录 `.env` 文件

#### 迁移步骤
```bash
# 1. 使用迁移脚本（推荐）
./scripts/migrate-backend-env.sh

# 2. 或手动迁移
cp .env.example .env
# 然后手动合并旧配置

# 3. 验证配置
./scripts/verify-env-cleanup.sh

# 4. 重新构建
docker-compose down
docker-compose build
docker-compose up -d
```

### 📊 影响范围

#### 修改的文件
- `src/backend/internal/controllers/slurm_controller.go`
- `src/backend/Dockerfile`
- `build.sh`
- `.env.example`
- `src/backend/.env.example`

#### 新增的文件
- `scripts/migrate-backend-env.sh`
- `scripts/verify-env-cleanup.sh`
- `src/backend/README_ENV.md`
- `docs/SALTSTACK_INTEGRATION_ENV_CONFIG.md`
- `docs/ENV_CLEANUP_REPORT.md`

#### 删除的文件
- `src/frontend/.env`
- `src/backend/.env`

### ✅ 验证结果

```
✓ 严重问题: 0
✓ 警告: 0

✓ 环境变量归一化验证通过！
所有环境变量已正确归一化到项目根目录 .env
```

### 📚 相关文档

- [环境变量归一化完成报告](./docs/ENV_CLEANUP_REPORT.md)
- [Backend 环境变量配置](./src/backend/README_ENV.md)
- [SaltStack 集成配置](./docs/SALTSTACK_INTEGRATION_ENV_CONFIG.md)

---

**维护人员**: AI Infrastructure Team  
**完成日期**: 2024-01-XX
