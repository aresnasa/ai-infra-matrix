# SLURM Task ID 字段类型修复完整指南

## 问题描述
数据库中 `slurm_tasks.task_id` 字段被 GORM 错误创建为 `bigint` 类型，导致无法插入 UUID 字符串。

错误信息：
```
ERROR: invalid input syntax for type bigint: "bf821ff2-5b39-4cec-b6be-f3543e571e3b" (SQLSTATE 22P02)
```

## 根本原因
1. GORM AutoMigrate 在某些情况下会错误推断字段类型
2. backend-init 容器使用的是旧版本代码，未包含最新修复

## 已完成的代码修复

### 1. 模型定义修复 (`src/backend/internal/models/slurm_task.go`)
```go
TaskID string `json:"task_id" gorm:"uniqueIndex;type:varchar(36);not null"` // UUID
```
明确指定 `type:varchar(36)`

### 2. 自动修复函数 (`src/backend/internal/database/database.go`)
添加 `fixSlurmTasksTableSchema()` 函数：
- 检查表是否存在
- 检查 task_id 字段类型
- 如果是 bigint，自动修复为 varchar(36)
- 重建唯一索引

### 3. 双重修复策略
在 `Migrate()` 函数中：
```go
// AutoMigrate 之前修复
fixSlurmTasksTableSchema()

// 执行 AutoMigrate
DB.AutoMigrate(...)

// AutoMigrate 之后再次修复（确保万无一失）
fixSlurmTasksTableSchema()
```

## 执行修复步骤

### 方法1：使用自动化脚本（推荐）

```bash
chmod +x scripts/rebuild-and-fix-complete.sh
./scripts/rebuild-and-fix-complete.sh
```

这个脚本会：
1. 停止 backend 和 backend-init 服务
2. 删除旧容器和镜像
3. 使用 `--no-cache` 重新构建 backend-init
4. 运行数据库初始化
5. 验证修复结果
6. 重新构建并启动 backend

### 方法2：手动执行（逐步控制）

```bash
# 1. 停止服务
docker-compose stop backend backend-init

# 2. 删除旧容器
docker-compose rm -f backend-init backend

# 3. 删除旧镜像（强制重新构建）
docker rmi ai-infra-backend-init:v0.3.6-dev

# 4. 重新构建 backend-init
docker-compose build --no-cache backend-init

# 5. 运行初始化
docker-compose up backend-init

# 6. 验证结果
docker-compose exec postgres psql -U postgres -d ai_infra_matrix -c "\d slurm_tasks" | grep task_id

# 7. 重新构建并启动 backend
docker-compose build backend
docker-compose up -d backend
```

### 方法3：使用 build.sh（项目标准流程）

```bash
# 重新构建 backend-init 和 backend
./build.sh build backend-init,backend --force

# 重新运行初始化
docker-compose stop backend-init
docker-compose rm -f backend-init
docker-compose up backend-init

# 重启 backend
docker-compose up -d backend
```

## 验证修复

### 1. 检查字段类型
```bash
docker-compose exec postgres psql -U postgres -d ai_infra_matrix -c "
SELECT column_name, data_type, character_maximum_length
FROM information_schema.columns 
WHERE table_name = 'slurm_tasks' AND column_name = 'task_id'
"
```

期望输出：
```
 column_name |     data_type     | character_maximum_length 
-------------+-------------------+--------------------------
 task_id     | character varying |                       36
```

### 2. 查看初始化日志
```bash
docker-compose logs backend-init | grep -i "slurm\|task_id\|fix"
```

期望看到：
```
level=info msg="Post-migration: fixing slurm_tasks table schema..."
level=info msg="Current task_id column type: bigint"
level=info msg="Fixing task_id column type from bigint to varchar(36)..."
level=info msg="✓ Successfully fixed task_id column type to varchar(36)"
```

或者（如果已经正确）：
```
level=info msg="✓ task_id column type is already varchar, no fix needed"
```

### 3. 测试创建任务
```bash
# 通过前端界面测试扩容功能
# 或使用 curl
curl -X POST http://localhost:8080/api/v1/slurm/scale-up/async \
  -H "Content-Type: application/json" \
  -d '{
    "partition": "compute",
    "node_count": 3,
    "node_prefix": "node"
  }'
```

应该不再出现 bigint 错误。

## 如果问题仍然存在

### 1. 检查镜像构建时间
```bash
docker images | grep backend-init
```
确保镜像是刚刚构建的（几分钟前）。

### 2. 检查模型定义
```bash
docker-compose exec backend cat /go/src/github.com/aresnasa/ai-infra-matrix/src/backend/internal/models/slurm_task.go | grep -A 1 "TaskID"
```
应该看到 `type:varchar(36)`

### 3. 手动修复数据库
如果自动修复失败，可以手动执行：
```bash
docker-compose exec postgres psql -U postgres -d ai_infra_matrix <<EOF
BEGIN;
TRUNCATE TABLE slurm_task_events CASCADE;
TRUNCATE TABLE slurm_tasks CASCADE;
DROP INDEX IF EXISTS idx_slurm_tasks_task_id;
ALTER TABLE slurm_tasks ALTER COLUMN task_id TYPE VARCHAR(36);
CREATE UNIQUE INDEX idx_slurm_tasks_task_id ON slurm_tasks(task_id);
COMMIT;
EOF
```

然后重启 backend：
```bash
docker-compose restart backend
```

## 常见问题

**Q: 为什么需要删除镜像后重新构建？**
A: Docker 会缓存构建层，即使代码更新了，如果 Dockerfile 没变，可能仍使用缓存的旧层。使用 `--no-cache` 或删除镜像可以强制完全重新构建。

**Q: 数据会丢失吗？**
A: 修复过程会清空 `slurm_tasks` 和 `slurm_task_events` 表，但这些是任务记录表，不影响其他数据。建议在生产环境前先备份。

**Q: 为什么需要在 AutoMigrate 前后都调用修复函数？**
A: 
- AutoMigrate 之前：修复已存在的表
- AutoMigrate 之后：修复 GORM 错误创建的表

**Q: 能否只重启容器而不重新构建？**
A: 不行。容器中运行的是镜像中的二进制文件，代码修改必须重新构建镜像。

## 技术细节

### GORM 类型推断问题
GORM 在处理 `string` 类型时，如果没有明确的 `type` 标签，可能会：
1. 检查数据库中是否有同名列
2. 尝试从现有数据推断类型
3. 在某些边缘情况下错误推断为数字类型

### PostgreSQL 类型兼容性
- `VARCHAR(36)`: 可变长度字符串，适合 UUID
- `BIGINT`: 64位整数，不能存储 UUID 字符串
- PostgreSQL 不会自动转换这两种类型

### UUID 格式
标准 UUID v4 格式：`xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx`
共36个字符（32个十六进制 + 4个连字符）

## 相关文件

- 模型定义：`src/backend/internal/models/slurm_task.go`
- 数据库迁移：`src/backend/internal/database/database.go`
- 初始化入口：`src/backend/cmd/init/main.go`
- Dockerfile：`src/backend/Dockerfile`
- 修复脚本：`scripts/rebuild-and-fix-complete.sh`
- SQL 脚本：`scripts/fix-slurm-tasks-table.sql`

## 修复历史

- 2025-10-26: 发现问题 - task_id 字段类型为 bigint
- 2025-10-26: 添加模型 type 标签
- 2025-10-26: 添加 fixSlurmTasksTableSchema() 函数
- 2025-10-26: 实现双重修复策略（AutoMigrate 前后）
- 2025-10-26: 创建自动化修复脚本

## 结论

通过以下三重保障机制，确保 task_id 字段类型正确：
1. **模型定义**：明确指定 `type:varchar(36)`
2. **前置修复**：AutoMigrate 前检查并修复
3. **后置修复**：AutoMigrate 后再次确认并修复

执行修复后，应该彻底解决 UUID 插入错误问题。
