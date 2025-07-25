# 数据库备份和恢复指南

本文档描述了Ansible Playbook Generator项目的数据库备份和恢复流程。

## 文件说明

### 核心文件

- `migrations/init_database.sql` - 完整的数据库初始化脚本，包含表结构和基础RBAC数据
- `db_manager.sh` - 数据库管理脚本，提供备份、恢复、初始化等功能
- `migrations/schema_backup.sql` - 当前数据库表结构备份
- `migrations/rbac_data_backup.sql` - RBAC相关表的数据备份

### 自动生成的备份文件

脚本会自动生成带时间戳的备份文件：
- `schema_backup_YYYYMMDD_HHMMSS.sql` - 表结构备份
- `rbac_data_backup_YYYYMMDD_HHMMSS.sql` - RBAC数据备份
- `full_backup_YYYYMMDD_HHMMSS.sql` - 完整数据库备份

## 使用方法

### 1. 初始化新数据库

```bash
# 首次部署时初始化数据库
cd /path/to/web-v2/backend
./db_manager.sh init
```

这将创建所有必要的表并插入基础的RBAC权限和角色数据，包括：
- 权限表：包含项目管理、用户管理等基础权限
- 角色表：super-admin（超级管理员）和user（普通用户）角色
- 角色权限关联：为角色分配相应的权限

### 2. 备份数据库

```bash
# 备份数据库结构
./db_manager.sh backup-schema

# 备份RBAC数据
./db_manager.sh backup-rbac

# 备份完整数据库（结构+数据）
./db_manager.sh backup-full
```

### 3. 恢复数据库

```bash
# 恢复数据库结构
./db_manager.sh restore-schema migrations/schema_backup_20250608_120000.sql

# 恢复RBAC数据
./db_manager.sh restore-data migrations/rbac_data_backup_20250608_120000.sql
```

### 4. 重置数据库

⚠️ **危险操作**：这将删除所有数据

```bash
# 完全重置数据库
./db_manager.sh reset
```

### 5. 查看帮助

```bash
./db_manager.sh help
```

## 数据库结构说明

### 核心表

1. **用户认证相关**
   - `users` - 用户表
   - `user_roles` - 用户角色关联表

2. **RBAC权限系统**
   - `permissions` - 权限表
   - `roles` - 角色表
   - `role_permissions` - 角色权限关联表
   - `user_groups` - 用户组表
   - `user_group_memberships` - 用户组成员关联表
   - `user_group_roles` - 用户组角色关联表
   - `role_bindings` - 角色绑定表

3. **业务数据**
   - `projects` - 项目表
   - `hosts` - 主机表
   - `tasks` - 任务表
   - `variables` - 变量表
   - `playbook_generations` - Playbook生成记录表

### 基础权限数据

初始化脚本会创建以下基础权限：

| 权限 | 描述 |
|------|------|
| `*:*:*` | 超级管理员权限 |
| `projects:create:*` | 创建项目权限 |
| `projects:read:*` | 查看项目权限 |
| `projects:update:*` | 更新项目权限 |
| `projects:delete:*` | 删除项目权限 |
| `projects:list:*` | 列出项目权限 |
| `users:*:*` | 用户管理权限 |
| `roles:*:*` | 角色管理权限 |
| `groups:*:*` | 用户组管理权限 |

### 基础角色配置

1. **super-admin（超级管理员）**
   - 拥有 `*:*:*` 权限，可以执行所有操作

2. **user（普通用户）**
   - 拥有项目相关的完整权限：
     - `projects:create:*`
     - `projects:read:*`
     - `projects:update:*`
     - `projects:delete:*`
     - `projects:list:*`

## 最佳实践

### 1. 定期备份

建议定期执行完整备份：

```bash
# 设置定时任务（crontab）
# 每天凌晨2点备份
0 2 * * * cd /path/to/web-v2/backend && ./db_manager.sh backup-full
```

### 2. 升级前备份

在升级应用前，务必执行完整备份：

```bash
./db_manager.sh backup-full
```

### 3. 开发环境重置

开发过程中如需重置数据库：

```bash
# 先备份当前状态（可选）
./db_manager.sh backup-full

# 重置数据库
./db_manager.sh reset
```

### 4. 生产环境迁移

从开发环境迁移到生产环境：

1. 在开发环境导出数据：
   ```bash
   ./db_manager.sh backup-full
   ```

2. 在生产环境初始化：
   ```bash
   ./db_manager.sh init
   ```

3. 根据需要恢复特定数据

## 故障排除

### 常见问题

1. **容器未运行**
   ```bash
   docker-compose up -d postgres
   ```

2. **权限不足**
   ```bash
   chmod +x db_manager.sh
   ```

3. **数据库连接失败**
   - 检查Docker容器状态
   - 确认数据库配置参数
   - 查看容器日志：`docker logs ansible-postgres`

### 日志查看

```bash
# 查看PostgreSQL容器日志
docker logs ansible-postgres

# 查看应用日志
docker logs ansible-backend
```

## 注意事项

1. **生产环境操作**：在生产环境执行任何数据库操作前，请务必备份
2. **权限管理**：确保脚本有足够的执行权限
3. **磁盘空间**：备份前检查磁盘空间是否足够
4. **网络连接**：确保Docker容器间网络连接正常
5. **数据一致性**：恢复操作可能需要停止应用服务以保证数据一致性

## 安全建议

1. **备份加密**：对敏感环境的备份文件进行加密存储
2. **访问控制**：限制备份文件的访问权限
3. **定期测试**：定期测试恢复流程的有效性
4. **版本控制**：将关键的初始化脚本纳入版本控制
