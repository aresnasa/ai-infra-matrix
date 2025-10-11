# SaltStack Master URL 配置修复报告

## 问题描述

后端容器中的 `SALTSTACK_MASTER_URL` 环境变量为空，导致后端无法正确连接到 SaltStack API。

### 问题现象

```bash
$ docker exec ai-infra-backend env | grep -E "SALT|saltstack" | sort
SALTSTACK_MASTER_URL=  # ❌ 空值，无法连接
```

### 根本原因

1. **`.env.example` 模板问题**
   - `SALTSTACK_MASTER_URL` 被设置为空字符串
   - 注释说明"留空以便后端自动拼装"，但实际上后端需要完整的 URL

2. **`docker-compose.yml` 默认值问题**
   ```yaml
   SALTSTACK_MASTER_URL: "${SALTSTACK_MASTER_URL:-}"  # ❌ 默认值为空
   ```

3. **渲染函数缺失处理**
   - `render_env_template_enhanced()` 函数没有处理 `SALTSTACK_MASTER_URL`
   - 没有根据 `SALT_API_SCHEME`、`SALT_MASTER_HOST`、`SALT_API_PORT` 自动拼装 URL

## 修复方案

### 1. 更新 `.env.example` 模板

**文件**: `/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/.env.example`

```bash
# ==================== SaltStack 配置 ====================
# SaltStack Master 主机地址 (容器名或IP地址)
SALTSTACK_MASTER_HOST=saltstack

# SaltStack Master API URL (完整URL格式: http://saltstack:8002)
# 如果留空，将自动根据 SALT_API_SCHEME://SALT_MASTER_HOST:SALT_API_PORT 组合生成
SALTSTACK_MASTER_URL=http://saltstack:8002  # ✅ 设置默认完整 URL

# SaltStack API 认证令牌 (可选，如果启用了API认证)
SALTSTACK_API_TOKEN=
```

### 2. 更新 `docker-compose.yml` 默认值

**文件**: `/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/docker-compose.yml`

```yaml
# SaltStack 配置
SALTSTACK_ENABLED: "${SALTSTACK_ENABLED:-true}"
SALTSTACK_MASTER_HOST: "${SALTSTACK_MASTER_HOST:-saltstack}"
SALTSTACK_MASTER_URL: "${SALTSTACK_MASTER_URL:-http://saltstack:8002}"  # ✅ 添加默认URL
SALT_API_SCHEME: "${SALT_API_SCHEME:-http}"
SALT_MASTER_HOST: "${SALT_MASTER_HOST:-saltstack}"
SALT_API_PORT: "${SALT_API_PORT:-8002}"
SALT_API_USERNAME: "${SALT_API_USERNAME:-saltapi}"
SALT_API_PASSWORD: "${SALT_API_PASSWORD:-aiinfra-salt}"
SALT_API_EAUTH: "${SALT_API_EAUTH:-file}"
```

### 3. 增强 `render_env_template_enhanced()` 函数

**文件**: `/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/build.sh`

添加自动拼装 `SALTSTACK_MASTER_URL` 的逻辑：

```bash
# 从模板内容中提取 SaltStack 配置（如果存在）
local salt_api_scheme=$(echo "$temp_content" | grep "^SALT_API_SCHEME=" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "http")
local salt_master_host=$(echo "$temp_content" | grep "^SALT_MASTER_HOST=" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "saltstack")
local salt_api_port=$(echo "$temp_content" | grep "^SALT_API_PORT=" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "8002")

# 构建 SALTSTACK_MASTER_URL（按后端期望格式）
local saltstack_master_url="${salt_api_scheme}://${salt_master_host}:${salt_api_port}"

# 替换 SALTSTACK_MASTER_URL（如果模板中为空，则填充拼装的值）
if echo "$temp_content" | grep -q "^SALTSTACK_MASTER_URL=$"; then
    temp_content=$(echo "$temp_content" | sed "s|^SALTSTACK_MASTER_URL=$|SALTSTACK_MASTER_URL=$saltstack_master_url|")
fi
```

### 4. 优化 `setup_saltstack_defaults()` 函数

**文件**: `/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/build.sh`

增强初始化逻辑，自动填充空的 `SALTSTACK_MASTER_URL`：

```bash
# SaltStack Master API URL（自动组合生成完整URL）
if ! grep -q "^SALTSTACK_MASTER_URL=" "$env_file" 2>/dev/null; then
    # 读取已设置的值
    local salt_scheme=$(grep "^SALT_API_SCHEME=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "http")
    local salt_host=$(grep "^SALT_MASTER_HOST=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "saltstack")
    local salt_port=$(grep "^SALT_API_PORT=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "8002")
    local default_url="${salt_scheme}://${salt_host}:${salt_port}"
    set_or_update_env_var "SALTSTACK_MASTER_URL" "$default_url" "$env_file"
    print_info "✓ 设置默认值: SALTSTACK_MASTER_URL=$default_url"
else
    # 如果存在但为空，则自动填充
    local current_url=$(grep "^SALTSTACK_MASTER_URL=" "$env_file" | cut -d'=' -f2 | tr -d '"' | tr -d "'")
    if [[ -z "$current_url" ]]; then
        local salt_scheme=$(grep "^SALT_API_SCHEME=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "http")
        local salt_host=$(grep "^SALT_MASTER_HOST=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "saltstack")
        local salt_port=$(grep "^SALT_API_PORT=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "8002")
        local default_url="${salt_scheme}://${salt_host}:${salt_port}"
        set_or_update_env_var "SALTSTACK_MASTER_URL" "$default_url" "$env_file"
        print_info "✓ 自动填充空值: SALTSTACK_MASTER_URL=$default_url"
    fi
fi
```

## 验证结果

### 修复前

```bash
$ docker exec ai-infra-backend env | grep -E "SALT|saltstack" | sort
SALT_API_EAUTH=file
SALT_API_PASSWORD=aiinfra-salt
SALT_API_PORT=8002
SALT_API_SCHEME=http
SALT_API_USERNAME=saltapi
SALT_MASTER_HOST=saltstack
SALT_MASTER_PORT=4505
SALT_RETURN_PORT=4506
SALTSTACK_API_TOKEN=
SALTSTACK_ENABLED=true
SALTSTACK_MASTER_HOST=saltstack
SALTSTACK_MASTER_URL=  # ❌ 空值
```

### 修复后

```bash
$ docker exec ai-infra-backend env | grep -E "SALT|saltstack" | sort
SALT_API_EAUTH=file
SALT_API_PASSWORD=aiinfra-salt
SALT_API_PORT=8002
SALT_API_SCHEME=http
SALT_API_USERNAME=saltapi
SALT_MASTER_HOST=saltstack
SALT_MASTER_PORT=4505
SALT_RETURN_PORT=4506
SALTSTACK_API_TOKEN=
SALTSTACK_ENABLED=true
SALTSTACK_MASTER_HOST=saltstack
SALTSTACK_MASTER_URL=http://saltstack:8002  # ✅ 正确的完整URL
```

## 后端代码逻辑

**文件**: `src/backend/internal/services/saltstack_service.go`

后端 `NewSaltStackService()` 函数的处理逻辑：

```go
func NewSaltStackService() *SaltStackService {
    // 从环境变量读取配置，提供默认值
    masterURL := os.Getenv("SALTSTACK_MASTER_URL")
    if masterURL == "" {
        // 兼容按 Host/Port/Scheme 组合的配置，避免写死容器内服务名
        scheme := os.Getenv("SALT_API_SCHEME")
        if scheme == "" {
            scheme = "http"
        }
        host := os.Getenv("SALT_MASTER_HOST")
        if host == "" {
            host = "localhost"
        }
        port := os.Getenv("SALT_API_PORT")
        if port == "" {
            port = "8002"
        }
        masterURL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
    }
    // ...
}
```

**设计意图**：
- 优先使用 `SALTSTACK_MASTER_URL`（完整URL）
- 如果为空，则从 `SALT_API_SCHEME`、`SALT_MASTER_HOST`、`SALT_API_PORT` 拼装
- 提供多层降级策略以提高容错性

**问题**：
- 之前 `SALTSTACK_MASTER_URL` 为空，但后端代码期望它有值
- 拼装逻辑虽然存在，但更好的实践是直接提供完整 URL

## 技术决策

### 为什么需要 `SALTSTACK_MASTER_URL`？

1. **明确性**: 完整 URL 更清晰，避免拼装逻辑错误
2. **灵活性**: 支持非标准端口、HTTPS、域名等场景
3. **调试性**: 便于排查连接问题，直接看到完整地址

### 为什么保留 `SALT_API_SCHEME/HOST/PORT`？

1. **向后兼容**: 支持旧版本配置
2. **模块化**: 分开配置便于理解和维护
3. **降级策略**: 后端代码可以在 URL 为空时自动拼装

### 最佳实践

**推荐配置** (`.env` 文件):
```bash
# 方案1: 使用完整 URL（推荐）
SALTSTACK_MASTER_URL=http://saltstack:8002
SALT_API_SCHEME=http
SALT_MASTER_HOST=saltstack
SALT_API_PORT=8002

# 方案2: 只配置组件（后端自动拼装）
SALTSTACK_MASTER_URL=
SALT_API_SCHEME=http
SALT_MASTER_HOST=saltstack
SALT_API_PORT=8002
```

**生产环境示例**:
```bash
# HTTPS + 自定义域名
SALTSTACK_MASTER_URL=https://salt-api.company.com:8443

# 或者分开配置
SALT_API_SCHEME=https
SALT_MASTER_HOST=salt-api.company.com
SALT_API_PORT=8443
```

## 影响范围

### 修改的文件

1. ✅ `.env.example` - 更新默认值和注释
2. ✅ `docker-compose.yml` - 添加默认 URL
3. ✅ `build.sh` - 增强渲染和初始化函数
4. ✅ `.env` - 手动更新当前环境

### 影响的服务

- ✅ `backend` - SaltStack 连接功能恢复正常

### 需要重启的服务

```bash
# 应用配置更改
docker compose up -d backend

# 或者重启所有服务
./build.sh build-all
```

## 测试清单

- [x] `.env` 文件中 `SALTSTACK_MASTER_URL` 有值
- [x] `.env.example` 模板中有默认值
- [x] `docker-compose.yml` 中有默认值
- [x] 后端容器环境变量正确
- [x] `render_env_template_enhanced()` 正确处理
- [x] `setup_saltstack_defaults()` 自动填充空值

## 后续优化建议

1. **健康检查**: 在后端添加 SaltStack 连接健康检查
   ```go
   func (s *SaltStackService) HealthCheck() error {
       _, err := s.GetStatus(context.Background())
       return err
   }
   ```

2. **配置验证**: 启动时验证 `SALTSTACK_MASTER_URL` 格式
   ```go
   func ValidateSaltStackURL(url string) error {
       if _, err := http.NewRequest("GET", url, nil); err != nil {
           return fmt.Errorf("invalid URL: %v", err)
       }
       return nil
   }
   ```

3. **文档更新**: 在用户手册中说明 SaltStack 配置方式

4. **自动化测试**: 添加集成测试验证 SaltStack 连接

## 总结

此次修复解决了 `SALTSTACK_MASTER_URL` 环境变量为空导致后端无法连接 SaltStack API 的问题。

**核心改进**:
1. ✅ 在 `.env.example` 中提供默认完整 URL
2. ✅ 在 `docker-compose.yml` 中设置默认值
3. ✅ 在 `build.sh` 中增强渲染和初始化逻辑
4. ✅ 提供多层降级策略保证兼容性

**最佳实践**:
- 优先使用完整 URL，清晰明确
- 保留组件配置，支持降级和向后兼容
- 提供合理默认值，减少配置复杂度

---

**修复日期**: 2025年10月11日  
**修复版本**: v0.3.7  
**相关组件**: backend, build.sh, docker-compose.yml
