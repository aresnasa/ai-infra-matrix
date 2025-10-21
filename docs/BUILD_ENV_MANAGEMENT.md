# Build.sh 环境变量管理优化说明

## 📋 优化内容

### 1. 移除硬编码，支持灵活配置

**之前的问题**：
- `render_env_template_enhanced` 函数硬编码了 Salt API 端口默认值 `8000`
- 检查脚本硬编码了端口号判断逻辑
- 不便于生产环境改造和自定义配置

**优化后**：
```bash
# 优先级顺序：环境变量 > 模板文件 > 默认值
local salt_api_port="${SALT_API_PORT:-$(echo "$temp_content" | grep "^SALT_API_PORT=" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "8000")}"
```

### 2. 环境变量优先级机制

**构建时的优先级（从高到低）**：

1. **当前 Shell 环境变量**（最高优先级）
   ```bash
   export SALT_API_PORT=8888
   ./build.sh build-all
   # 使用端口 8888
   ```

2. **.env 文件中的值**
   ```bash
   SALT_API_PORT=9000
   ```

3. **.env.example 文件中的默认值**
   ```bash
   SALT_API_PORT=8000
   ```

4. **代码中的回退默认值**（最低优先级）
   ```bash
   echo "8000"  # 仅当前面都未配置时使用
   ```

### 3. 配置同步策略

`sync_env_with_example` 函数行为：

| 情况 | 操作 | 原因 |
|------|------|------|
| .env 中**缺失**某配置项 | ✅ 自动添加 | 确保配置完整 |
| .env 中配置项**值为空** | ✅ 填充默认值 | 避免空配置 |
| .env 中配置项**已有值** | ⛔ 保持不变 | 保护用户自定义 |

### 4. 诊断脚本优化

**check-saltstack-config.sh** 不再硬编码端口：

```bash
# 之前：硬编码检查 8000
if [ "$ENV_PORT" = "8000" ]; then
    echo "✓ 配置正确"
fi

# 现在：动态对比 .env 和 .env.example
if [ "$ENV_PORT" = "$EXAMPLE_PORT" ]; then
    echo "✓ 配置与推荐值一致"
fi
```

## 🎯 使用场景

### 场景 1：开发环境（默认配置）

```bash
# 直接使用默认配置
./build.sh build-all
```

### 场景 2：生产环境（修改默认值）

```bash
# 1. 修改 .env.example（推荐方式）
vim .env.example
# 修改: SALT_API_PORT=8888

# 2. 运行构建（自动同步）
./build.sh build-all

# 3. .env 会自动同步新的默认值
```

### 场景 3：临时测试不同端口

```bash
# 不修改文件，直接用环境变量
export SALT_API_PORT=7777
export SALTSTACK_MASTER_URL=http://saltstack:7777
./build.sh build-all
```

### 场景 4：多环境配置

```bash
# 开发环境
cp .env.example .env.dev
vim .env.dev  # 修改为开发环境配置

# 生产环境
cp .env.example .env.prod
vim .env.prod  # 修改为生产环境配置

# 使用指定环境
ln -sf .env.dev .env
./build.sh build-all

# 切换环境
ln -sf .env.prod .env
./build.sh build-all
```

## 📝 最佳实践

### 1. 新项目初始化

```bash
# 1. 复制配置模板
cp .env.example .env

# 2. 修改必要配置（密码、域名等）
vim .env

# 3. 构建
./build.sh build-all
```

### 2. 生产环境部署

```bash
# 1. 修改 .env.example 为生产环境默认值
vim .env.example
# SALT_API_PORT=8000
# SALTSTACK_MASTER_URL=http://salt-master.prod.local:8000

# 2. 同步到 .env
./build.sh build-all
# 会提示新增或更新的配置项

# 3. 验证配置
./scripts/check-saltstack-config.sh
```

### 3. 配置更新

```bash
# 当 .env.example 更新后
./build.sh build-all

# 会提示：
# 新增的配置项 (2):
#   + SALT_API_TIMEOUT=8s
#   + SALT_API_EAUTH=file
#
# 更新的配置项 (1):
#   ↻ SALT_API_PORT (空值 → 8000)
#
# 保持不变的配置项 (50):
#   ✓ 50 个配置项值未更改（保留用户自定义值）
```

## 🔧 troubleshooting

### 问题 1：配置不生效

**现象**：修改了 .env 但服务还是用旧配置

**原因**：
1. 容器使用的是镜像构建时的配置
2. 需要重新构建镜像

**解决**：
```bash
# 方案 1：重新构建
./build.sh build-all

# 方案 2：只重启（如果是运行时配置）
docker-compose restart backend

# 方案 3：使用环境变量覆盖
export SALT_API_PORT=8888
docker-compose up -d backend
```

### 问题 2：端口冲突

**现象**：Salt Master 连接超时

**诊断**：
```bash
# 1. 检查配置
./scripts/check-saltstack-config.sh

# 2. 检查实际监听端口
docker-compose exec saltstack netstat -tlnp | grep salt-api

# 3. 检查容器日志
docker-compose logs saltstack | grep -i port
```

**解决**：
```bash
# 修改为实际端口
vim .env
# SALT_API_PORT=实际端口

# 重启后端
docker-compose restart backend
```

### 问题 3：不知道该配置什么端口

**查看 Salt Master 实际配置**：
```bash
# 进入容器
docker-compose exec saltstack bash

# 查看 salt-api 配置
cat /etc/salt/master.d/api.conf

# 查看监听端口
netstat -tlnp | grep python
```

**参考 .env.example**：
```bash
# 查看推荐配置
grep SALT .env.example
```

## 📊 配置验证清单

运行构建前检查：

- [ ] `.env.example` 是否包含所有必要配置
- [ ] `.env` 文件是否存在
- [ ] 敏感信息（密码）是否已修改
- [ ] Salt API 端口是否与实际一致
- [ ] 外部访问地址是否正确

运行构建后验证：

- [ ] 运行 `./scripts/check-saltstack-config.sh`
- [ ] 检查 `docker-compose logs backend` 无错误
- [ ] 访问 `http://<host>:8080/saltstack` 测试
- [ ] 运行 Playwright 测试验证功能

## 🎓 总结

**核心改进**：
1. ✅ 移除所有硬编码端口
2. ✅ 支持多级配置优先级
3. ✅ 自动同步配置更新
4. ✅ 保护用户自定义配置
5. ✅ 灵活支持多环境部署

**设计原则**：
- **约定优于配置**：提供合理的默认值
- **显式优于隐式**：清晰的优先级规则
- **保护用户数据**：不覆盖已有配置
- **便于维护**：集中管理配置模板

**适用场景**：
- 开发环境快速搭建
- 生产环境标准化部署
- 多环境配置管理
- 临时测试和调试

---

**维护人员**: AI Infrastructure Team  
**更新日期**: 2025-10-21  
**相关文件**:
- `build.sh` - 主构建脚本
- `.env.example` - 配置模板
- `scripts/check-saltstack-config.sh` - 配置诊断工具
